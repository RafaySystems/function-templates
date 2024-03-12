package sdk

// adapted from https://github.com/openfaas/golang-http-template/blob/master/template/golang-http/main.go
// Original license: MIT

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	httputil "github.com/RafaySystems/envmgr-pkgs/http"
)

var (
	acceptingConnections int32
)

type SDKOptions struct {
	Port                int
	Handler             Handler
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	ShutdownTimeout     time.Duration
	HealthInterval      time.Duration
	LogLevel            slog.Level
	LogUploadRetryCount int
}

type SDKOption func(*SDKOptions)

func WithPort(port int) SDKOption {
	return func(o *SDKOptions) {
		o.Port = port
	}
}

func WithHandler(handler Handler) SDKOption {
	return func(o *SDKOptions) {
		o.Handler = handler
	}
}

func WithReadTimeout(readTimeout time.Duration) SDKOption {
	return func(o *SDKOptions) {
		o.ReadTimeout = readTimeout
	}
}

func WithWriteTimeout(writeTimeout time.Duration) SDKOption {
	return func(o *SDKOptions) {
		o.WriteTimeout = writeTimeout
	}
}

func WithHealthInterval(healthInterval time.Duration) SDKOption {
	return func(o *SDKOptions) {
		o.HealthInterval = healthInterval
	}
}

func WithLogLevel(logLevel slog.Level) SDKOption {
	return func(o *SDKOptions) {
		o.LogLevel = logLevel
	}

}

func WithLogUploadRetryCount(logUploadRetryCount int) SDKOption {
	return func(o *SDKOptions) {
		o.LogUploadRetryCount = logUploadRetryCount
	}
}

func WithShutdownTimeout(shutdownTimeout time.Duration) SDKOption {
	return func(o *SDKOptions) {
		o.ShutdownTimeout = shutdownTimeout
	}
}

func NewFunctionSDK(opts ...SDKOption) (*FunctionSDK, error) {
	options := &SDKOptions{
		Port:                5000,
		Handler:             nil,
		ReadTimeout:         10 * time.Second,
		WriteTimeout:        10 * time.Second,
		HealthInterval:      10 * time.Second,
		LogLevel:            slog.LevelInfo,
		LogUploadRetryCount: 3,
		ShutdownTimeout:     10 * time.Second,
	}

	for _, o := range opts {
		o(options)
	}

	if options.Handler == nil {
		return nil, fmt.Errorf("Handler is required")
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	})

	logger := slog.New(handler)

	return &FunctionSDK{
		logger:          logger,
		port:            options.Port,
		handler:         options.Handler,
		readTimeout:     options.ReadTimeout,
		writeTimeout:    options.WriteTimeout,
		healthInterval:  options.HealthInterval,
		logLevel:        options.LogLevel,
		shutdownTimeout: options.ShutdownTimeout,
		client:          httputil.NewRetriableHTTPClient(httputil.WithMaxRetryCount(options.LogUploadRetryCount)).StandardClient(),
	}, nil

}

type FunctionSDK struct {
	logger          *slog.Logger
	port            int
	handler         Handler
	readTimeout     time.Duration
	writeTimeout    time.Duration
	healthInterval  time.Duration
	logLevel        slog.Level
	client          *http.Client
	shutdownTimeout time.Duration
}

func (fsdk *FunctionSDK) Run(ctx context.Context) error {

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", fsdk.port))
	if err != nil {
		return err
	}

	errChan := make(chan error, 1)
	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", fsdk.port),
		ReadTimeout:    fsdk.readTimeout,
		WriteTimeout:   fsdk.writeTimeout,
		MaxHeaderBytes: 1 << 20, // Max header of 1MB
	}

	http.HandleFunc("/", fsdk.getFunctionHandler())
	http.HandleFunc("/_/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")

		var resp = ReadyResponse{
			Ready:          true,
			NumConnections: atomic.LoadInt32(&acceptingConnections),
		}

		json.NewEncoder(w).Encode(resp)
	})

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := s.Serve(listener); err != http.ErrServerClosed {
			fsdk.logger.Error("[entrypoint] Error Serve", "error", err)
			errChan <- err
		}
	}()

	atomic.StoreInt32(&acceptingConnections, 1)

	go func() {
		defer wg.Done()
		<-ctx.Done()
		atomic.StoreInt32(&acceptingConnections, 0)
		shutdownctx, cancel := context.WithTimeout(context.Background(), fsdk.shutdownTimeout)
		defer cancel()
		err := s.Shutdown(shutdownctx)
		if err != nil {
			fsdk.logger.Error("[entrypoint] Error in Shutdown", "error", err)
			errChan <- err
		}
	}()

	wg.Wait()

	return nil
}

func (fsdk *FunctionSDK) getFunctionHandler() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		piper, pipew := io.Pipe()
		defer pipew.Close()

		logger := slog.New(slog.NewTextHandler(pipew, &slog.HandlerOptions{
			AddSource: true,
			Level:     fsdk.logLevel,
		}))

		activityID := r.Header.Get(ActivityIDHeader)
		environmentID := r.Header.Get(EnvironmentIDHeader)
		environmentName := r.Header.Get(EnvironmentNameHeader)
		engineEndpoint := r.Header.Get(EngineAPIEndpointHeader)
		fileUploadPath := r.Header.Get(ActivityFileUploadHeader)

		logger.With("activityID", activityID).
			With("environmentID", environmentID).
			With("environmentName", environmentName)

		path := engineEndpoint + fileUploadPath

		req, err := http.NewRequestWithContext(r.Context(), "POST", path, piper)
		if err != nil {
			logger.Info("Error creating request", "error", err, "path", path)
			return
		}

		handler := fsdk.makeRequestHandler(logger)
		handler(w, r)

		req.Header.Add(WorkflowTokenHeader, r.Header.Get(WorkflowTokenHeader))

		resp, err := fsdk.client.Do(req)
		if err != nil {
			logger.Info("Error in file upload", "error", err)
			return
		}

		defer resp.Body.Close()
	}

}

func (fsdk *FunctionSDK) makeRequestHandler(logger *slog.Logger) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var input []byte

		if r.Body != nil {
			defer r.Body.Close()

			bodyBytes, bodyErr := io.ReadAll(r.Body)

			if bodyErr != nil {
				log.Printf("Error reading body from request.")
			}

			input = bodyBytes
		}

		var req Request
		err := json.Unmarshal(input, &req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid input"))
			return
		}

		if req == nil {
			req = make(Request)
		}
		req["metadata"] = map[string]string{
			"activityID":      r.Header.Get(ActivityIDHeader),
			"environmentID":   r.Header.Get(EnvironmentIDHeader),
			"environmentName": r.Header.Get(EnvironmentNameHeader),
		}

		result, err := fsdk.handler(r.Context(), logger, req)
		if err != nil {
			logger.Error("Error in function", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		err = json.NewEncoder(w).Encode(result)
		if err != nil {
			logger.Error("Error in encoding response", "error", err)
		}
	}
}
