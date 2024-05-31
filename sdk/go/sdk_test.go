package sdk_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sdk "github.com/RafaySystems/function-templates/sdk/go"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestFunctionSDK(t *testing.T) {

	testcases := map[string]struct {
		handler    func(context.Context, sdk.Logger, sdk.Request) (sdk.Response, error)
		statusCode int
		response   sdk.Response
		err        sdk.ErrFunction
	}{
		"success": {
			handler: func(ctx context.Context, logger sdk.Logger, req sdk.Request) (sdk.Response, error) {
				logger.Info("Request received", "request", req)
				for i := 0; i < 10; i++ {
					logger.Info("Log message", "i", i)
					time.Sleep(1 * time.Second)
				}
				return sdk.Response{"output1": "value1"}, nil
			},
			response:   sdk.Response{"output1": "value1"},
			statusCode: http.StatusOK,
		},
		"panic": {
			handler: func(ctx context.Context, logger sdk.Logger, req sdk.Request) (sdk.Response, error) {
				logger.Info("Request received", "request", req)
				panic("panic")
			},
			err: sdk.ErrFunction{
				Message: "Panic in function: panic",
				ErrCode: sdk.ErrCodeFailed,
			},
			statusCode: http.StatusInternalServerError,
		},
		"general-error": {
			handler: func(ctx context.Context, logger sdk.Logger, req sdk.Request) (sdk.Response, error) {
				logger.Info("Request received", "request", req)
				return nil, fmt.Errorf("general error")
			},
			err: sdk.ErrFunction{
				Message: "general error",
				ErrCode: sdk.ErrCodeFailed,
			},
			statusCode: http.StatusInternalServerError,
		},
		"failed-error": {
			handler: func(ctx context.Context, logger sdk.Logger, req sdk.Request) (sdk.Response, error) {
				logger.Info("Request received", "request", req)
				return nil, fmt.Errorf("%w: %s", sdk.NewErrFailed("failed"), "wrapping error")
			},
			err: sdk.ErrFunction{
				Message: "failed: wrapping error",
				ErrCode: sdk.ErrCodeFailed,
			},
			statusCode: http.StatusInternalServerError,
		},
		"transient-error": {
			handler: func(ctx context.Context, logger sdk.Logger, req sdk.Request) (sdk.Response, error) {
				logger.Info("Request received", "request", req)
				return nil, sdk.NewErrTransient("transient")
			},
			err: sdk.ErrFunction{
				Message: "transient",
				ErrCode: sdk.ErrCodeTransient,
			},
			statusCode: http.StatusInternalServerError,
		},
		"execute-again-error": {
			handler: func(ctx context.Context, logger sdk.Logger, req sdk.Request) (sdk.Response, error) {
				logger.Info("Request received", "request", req)
				return nil, fmt.Errorf("%w: %s", sdk.NewErrExecuteAgain("execute again", map[string]interface{}{"key": "value"}), "wrapping error")
			},
			err: sdk.ErrFunction{
				Message: "execute again: wrapping error",
				ErrCode: sdk.ErrCodeExecuteAgain,
				Data:    map[string]interface{}{"key": "value"},
			},
			statusCode: http.StatusInternalServerError,
		},
	}

	logs := make(map[string][]byte)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			defer r.Body.Close()
			reader, err := r.MultipartReader()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			part, err := reader.NextPart()
			if err != nil {
				if err == io.EOF {
					return
				}
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer part.Close()
			bodyBytes, _ := io.ReadAll(part)
			logs[r.URL.Path] = append(logs[r.URL.Path], bodyBytes...)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", 0))
	if err != nil {
		t.Errorf("Error creating listener: %v", err)
		return
	}
	funcSDK, err := sdk.NewFunctionSDK(
		sdk.WithListener(listener),
		sdk.WithReadTimeout(20*time.Second),
		sdk.WithWriteTimeout(20*time.Second),
		sdk.WithHandler(
			func(ctx context.Context, logger sdk.Logger, req sdk.Request) (sdk.Response, error) {
				if key, ok := req["key"].(string); ok {
					if tc, ok := testcases[key]; ok {
						return tc.handler(ctx, logger, req)
					}
				}
				return nil, fmt.Errorf("handler not found")
			},
		),
	)
	if err != nil {
		t.Errorf("Error creating function SDK: %v", err)
		return
	}

	go func() {
		err := funcSDK.Run(context.Background())
		if err != nil {
			t.Errorf("Error running function SDK: %v", err)
		}
	}()

	for key, tc := range testcases {
		r := bytes.NewReader([]byte(fmt.Sprintf("{\"key\": \"%s\"}", key)))

		req, err := http.NewRequest("POST", fmt.Sprintf("http://%s", listener.Addr().String()), r)
		if err != nil {
			t.Errorf("Error creating request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(sdk.ActivityIDHeader, "activity1")
		req.Header.Set(sdk.EnvironmentIDHeader, "environment1")
		req.Header.Set(sdk.EnvironmentNameHeader, "environment1Name")
		req.Header.Set(sdk.EngineAPIEndpointHeader, server.URL)
		req.Header.Set(sdk.ActivityFileUploadHeader, fmt.Sprintf("/activity/%s/log", key))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("Error sending request: %v", err)
			return
		}

		if resp.StatusCode == http.StatusInternalServerError {
			if resp.Body != nil {
				defer resp.Body.Close()
				var functionErr sdk.ErrFunction
				err := json.NewDecoder(resp.Body).Decode(&functionErr)
				if err != nil {
					t.Errorf("Error decoding error response: %v", err)
				}
				if diff := cmp.Diff(tc.err, functionErr,
					cmpopts.IgnoreFields(sdk.ErrFunction{}, "StackTrace")); diff != "" {
					t.Errorf("Unexpected error response: %s", diff)
				}
				if strings.Contains(key, "panic") && len(functionErr.StackTrace) == 0 {
					t.Errorf("Expected stack trace in error response")
				}
			} else {
				t.Errorf("Unexpected empty response")
			}
		}
		if resp.StatusCode == http.StatusOK {
			if resp.Body != nil {
				defer resp.Body.Close()
				var result struct {
					Data sdk.Response `json:"data"`
				}
				err := json.NewDecoder(resp.Body).Decode(&result)
				if err != nil {
					t.Errorf("Error decoding response: %v", err)
				}
				if diff := cmp.Diff(tc.response, result.Data); diff != "" {
					t.Errorf("Unexpected response: %s", diff)
				}
			} else {
				t.Errorf("Unexpected empty response")
			}
		}
	}

	// check logs if empty
	for _, log := range logs {
		if len(log) == 0 {
			t.Errorf("Expected log to be non-empty")
		}
	}

}
