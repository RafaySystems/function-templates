package sdk

// adapted from https://github.com/openfaas/golang-http-template/blob/master/template/golang-http/main.go
// Original license: MIT

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
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
	Listener            net.Listener
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

func WithListener(listener net.Listener) SDKOption {
	return func(o *SDKOptions) {
		o.Listener = listener
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
		Listener:            nil,
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
		listener:        options.Listener,
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
	listener        net.Listener
	handler         Handler
	readTimeout     time.Duration
	writeTimeout    time.Duration
	healthInterval  time.Duration
	logLevel        slog.Level
	client          *http.Client
	shutdownTimeout time.Duration
}

func (fsdk *FunctionSDK) Run(ctx context.Context) error {

	var listener net.Listener
	var err error
	if fsdk.listener != nil {
		listener = fsdk.listener
	} else {
		listener, err = net.Listen("tcp", fmt.Sprintf(":%d", fsdk.port))
		if err != nil {
			return err
		}
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

		activityID := r.Header.Get(ActivityIDHeader)
		environmentID := r.Header.Get(EnvironmentIDHeader)
		environmentName := r.Header.Get(EnvironmentNameHeader)
		engineEndpoint := r.Header.Get(EngineAPIEndpointHeader)
		fileUploadPath := r.Header.Get(ActivityFileUploadHeader)

		piper, pipew := io.Pipe()
		writer := multipart.NewWriter(pipew)
		defer func() {
			writer.Close()
			pipew.Close()
			piper.Close()
		}()

		path := engineEndpoint + fileUploadPath

		req, err := http.NewRequestWithContext(r.Context(), "POST", path, piper)
		if err != nil {
			fsdk.logger.Info("Error creating request", "error", err, "path", path)
			return
		}

		req.Header.Add(WorkflowTokenHeader, r.Header.Get(WorkflowTokenHeader))
		req.Header.Add("Content-Type", writer.FormDataContentType())

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := fsdk.client.Do(req)
			if err != nil {
				fsdk.logger.Info("Error in file upload", "error", err)
				return
			}
			if resp.Body != nil {
				defer resp.Body.Close()
			}
		}()

		part, err := writer.CreateFormFile("content", "stdout")
		if err != nil {
			fsdk.logger.Info("Error creating form field for logging", "error", err)
			return
		}

		logger := slog.New(slog.NewTextHandler(part, &slog.HandlerOptions{
			AddSource: true,
			Level:     fsdk.logLevel,
		}))
		logger = logger.With("activityID", activityID).
			With("environmentID", environmentID).
			With("environmentName", environmentName)
		fsdk.logger.Info("invoking function", "activityID", activityID, "environmentID", environmentID, "environmentName", environmentName)
		logger.Info("invoking function")

		handler := fsdk.makeRequestHandler(logger)
		handler(w, r)

		writer.Close()
		pipew.Close()
		wg.Wait()
	}

}

func (fsdk *FunctionSDK) makeRequestHandler(logger *slog.Logger) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		var input []byte

		if r.Body != nil {
			defer r.Body.Close()

			bodyBytes, bodyErr := io.ReadAll(r.Body)

			if bodyErr != nil {
				logger.Error("Error reading body from request.")
			}

			input = bodyBytes
		}

		var req Request

		if len(input) > 0 {
			err := json.Unmarshal(input, &req)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid input"))
				return
			}
		}

		if req == nil {
			req = make(Request)
		}
		req["metadata"] = map[string]string{
			"activityID":      r.Header.Get(ActivityIDHeader),
			"environmentID":   r.Header.Get(EnvironmentIDHeader),
			"environmentName": r.Header.Get(EnvironmentNameHeader),
		}

		result, err := fsdk.invokeHandler(r.Context(), logger, req)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			if errFunc, ok := AsErrFunction(err); ok {
				err := json.NewEncoder(w).Encode(errFunc)
				if err != nil {
					logger.Error("Error in encoding error response", "error", err)
				}
			} else {
				err := json.NewEncoder(w).Encode(
					&ErrFunction{Message: err.Error(), ErrCode: ErrCodeFailed},
				)
				if err != nil {
					logger.Error("Error in encoding error response", "error", err)
				}
			}
			return
		} else {
			w.WriteHeader(http.StatusOK)
		}

		err = json.NewEncoder(w).Encode(map[string]any{"data": result})
		if err != nil {
			logger.Error("Error in encoding response", "error", err)
		}
	}
}

func (fsdk *FunctionSDK) invokeHandler(ctx context.Context, logger *slog.Logger, req Request) (r Response, err error) {
	defer func() {
		if panic := recover(); panic != nil {
			logger.Error("Panic in function", "panic", panic)
			err = newErrFailedWithStackTrace(fmt.Sprintf("Panic in function: %v", panic))
		}
	}()
	r, err = fsdk.handler(ctx, logger, req)
	if err != nil {
		return nil, err
	}

	return r, nil
}
