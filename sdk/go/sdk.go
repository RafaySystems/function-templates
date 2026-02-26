package sdk

// adapted from https://github.com/openfaas/golang-http-template/blob/master/template/golang-http/main.go
// Original license: MIT

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	httputil "github.com/RafaySystems/envmgr-pkgs/http"
	slogmulti "github.com/samber/slog-multi"
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
	LogFlushRate        time.Duration
	LogWriteTimeout     time.Duration
	SkipTLSVerify       bool
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

func WithLogWriteTimeout(logWriteTimeout time.Duration) SDKOption {
	return func(o *SDKOptions) {
		o.LogWriteTimeout = logWriteTimeout
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

func WithLogFlushRate(logFlushRate time.Duration) SDKOption {
	return func(o *SDKOptions) {
		o.HealthInterval = logFlushRate
	}
}

func WithServerSkipTLSVerify(skipTLSVerify bool) SDKOption {
	return func(o *SDKOptions) {
		o.SkipTLSVerify = skipTLSVerify
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
		LogFlushRate:        1 * time.Second,
		SkipTLSVerify:       false,
	}

	for _, o := range opts {
		o(options)
	}

	if options.Handler == nil {
		return nil, fmt.Errorf("handler is required")
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
		logFlushRate:    options.LogFlushRate,
		logWriteTimeout: options.LogWriteTimeout,
		skipTLSVerify:   options.SkipTLSVerify,
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
	logFlushRate    time.Duration
	logWriteTimeout time.Duration
	skipTLSVerify   bool
}

func (f *FunctionSDK) Run(ctx context.Context) error {
	var listener net.Listener
	var err error
	if f.listener != nil {
		listener = f.listener
	} else {
		listener, err = net.Listen("tcp", fmt.Sprintf(":%d", f.port))
		if err != nil {
			return err
		}
	}

	errChan := make(chan error, 1)
	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", f.port),
		ReadTimeout:    f.readTimeout,
		WriteTimeout:   f.writeTimeout,
		MaxHeaderBytes: 1 << 20, // Max header of 1MB
	}

	http.HandleFunc("/", f.getFunctionHandler())
	http.HandleFunc("/_/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")

		var resp = ReadyResponse{
			Ready:          true,
			NumConnections: atomic.LoadInt32(&acceptingConnections),
		}

		_ = json.NewEncoder(w).Encode(resp)
	})

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := s.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			f.logger.Error("[entrypoint] Error Serve", "error", err)
			errChan <- err
		}
	}()

	atomic.StoreInt32(&acceptingConnections, 1)

	go func() {
		defer wg.Done()
		<-ctx.Done()
		atomic.StoreInt32(&acceptingConnections, 0)
		shutdownctx, cancel := context.WithTimeout(context.Background(), f.shutdownTimeout)
		defer cancel()
		err := s.Shutdown(shutdownctx)
		if err != nil {
			f.logger.Error("[entrypoint] Error in Shutdown", "error", err)
			errChan <- err
		}
	}()

	wg.Wait()

	return nil
}

func (f *FunctionSDK) getFunctionHandler() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		activityID := r.Header.Get(ActivityIDHeader)
		environmentID := r.Header.Get(EnvironmentIDHeader)
		environmentName := r.Header.Get(EnvironmentNameHeader)
		engineEndpoint := r.Header.Get(EngineAPIEndpointHeader)
		fileUploadPath := r.Header.Get(ActivityFileUploadHeader)

		currLogger := f.logger.With("activityID", activityID).
			With("environmentID", environmentID).
			With("environmentName", environmentName)

		url := engineEndpoint + fileUploadPath
		logWriter := NewActivityLogWriter(r.Context(), currLogger, url, r.Header.Get(WorkflowTokenHeader), WithLogReqTimeout(f.logWriteTimeout), WithWriteFlushTickRate(f.logFlushRate), WithSkipTLSVerify(f.skipTLSVerify))
		defer logWriter.Close()

		logger := slog.New(slogmulti.Fanout(slog.NewTextHandler(logWriter, &slog.HandlerOptions{
			AddSource: true,
			Level:     f.logLevel,
		}), currLogger.Handler()))
		logger.Info("invoking function")

		currLogger.Info("invoking function")

		handler := f.makeRequestHandler(logger)
		handler(w, r)
	}

}

func (f *FunctionSDK) makeRequestHandler(logger *slog.Logger) http.HandlerFunc {

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
				_, _ = w.Write([]byte("Invalid input"))
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
			"organizationID":  r.Header.Get(OrganizationIDHeader),
			"projectID":       r.Header.Get(ProjectIDHeader),
			"stateStoreUrl":   r.Header.Get(EaasStateEndpointHeader),
			"stateStoreToken": r.Header.Get(EaasStateAPITokenHeader),
			"eventSource":     r.Header.Get(EventSourceHeader),
			"eventSourceName": r.Header.Get(EventSourceNameHeader),
			"eventType":       r.Header.Get(EventTypeHeader),
		}

		result, err := f.invokeHandler(r.Context(), logger, req)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			errFunc, ok := AsErrFunction(err)
			if !ok {
				errFunc = &ErrFunction{Message: err.Error(), ErrCode: ErrCodeFailed}
			}
			if err = json.NewEncoder(w).Encode(errFunc); err != nil {
				logger.Error("Error in encoding error response", "error", err)
			}
			return
		}

		w.WriteHeader(http.StatusOK)
		if err = json.NewEncoder(w).Encode(map[string]any{"data": result}); err != nil {
			logger.Error("Error in encoding response", "error", err)
		}
	}
}

func (f *FunctionSDK) invokeHandler(ctx context.Context, logger *slog.Logger, req Request) (r Response, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			logger.Error("Panic in function", "panic", rec)
			err = newErrFailedWithStackTrace(fmt.Sprintf("Panic in function: %v", rec))
		}
	}()
	return f.handler(ctx, logger, req)
}
