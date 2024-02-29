// adapted from https://github.com/openfaas/golang-http-template/blob/master/template/golang-http/main.go
// Original license: MIT
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"handler/function"

	sdk "github.com/RafaySystems/function-templates/sdk/go"
)

var (
	acceptingConnections int32
)

type readyResponse struct {
	Ready          bool  `json:"ready"`
	NumConnections int32 `json:"num_connections"`
}

const defaultTimeout = 10 * time.Second

func main() {
	readTimeout := parseIntOrDurationValue(os.Getenv("read_timeout"), defaultTimeout)
	writeTimeout := parseIntOrDurationValue(os.Getenv("write_timeout"), defaultTimeout)
	healthInterval := parseIntOrDurationValue(os.Getenv("healthcheck_interval"), writeTimeout)

	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", 8082),
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		MaxHeaderBytes: 1 << 20, // Max header of 1MB
	}

	http.HandleFunc("/", makeRequestHandler())
	http.HandleFunc("/_/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")

		var resp = readyResponse{
			Ready:          true,
			NumConnections: atomic.LoadInt32(&acceptingConnections),
		}

		json.NewEncoder(w).Encode(resp)
	})

	listenUntilShutdown(s, healthInterval, writeTimeout)
}

func listenUntilShutdown(s *http.Server, shutdownTimeout time.Duration, writeTimeout time.Duration) {
	idleConnsClosed := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM)

		<-sig

		log.Printf("[entrypoint] SIGTERM: no connections in: %s", shutdownTimeout.String())
		<-time.Tick(shutdownTimeout)

		ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
		defer cancel()

		if err := s.Shutdown(ctx); err != nil {
			log.Printf("[entrypoint] Error in Shutdown: %v", err)
		}

		log.Printf("[entrypoint] Exiting.")

		close(idleConnsClosed)
	}()

	// Run the HTTP server in a separate go-routine.
	go func() {
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("[entrypoint] Error ListenAndServe: %v", err)
			close(idleConnsClosed)
		}
	}()

	atomic.StoreInt32(&acceptingConnections, 1)

	<-idleConnsClosed
}

func makeRequestHandler() func(http.ResponseWriter, *http.Request) {
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

		req := sdk.Request{
			Body:        input,
			Header:      r.Header,
			Method:      r.Method,
			QueryString: r.URL.RawQuery,
		}

		result, resultErr := function.Handle(r.Context(), req)

		if result.Header != nil {
			for k, v := range result.Header {
				w.Header()[k] = v
			}
		}

		if resultErr != nil {
			log.Print(resultErr)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			if result.StatusCode == 0 {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(result.StatusCode)
			}
		}

		w.Write(result.Body)
	}
}

func parseIntOrDurationValue(val string, fallback time.Duration) time.Duration {
	if len(val) > 0 {
		parsedVal, parseErr := strconv.Atoi(val)
		if parseErr == nil && parsedVal >= 0 {
			return time.Duration(parsedVal) * time.Second
		}
	}

	duration, durationErr := time.ParseDuration(val)
	if durationErr != nil {
		return fallback
	}
	return duration
}
