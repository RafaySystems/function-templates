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
	"testing"

	sdk "github.com/RafaySystems/function-templates/sdk/go"
)

func TestFunctionSDK(t *testing.T) {

	logs := make(map[string][]byte)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			defer r.Body.Close()
			bodyBytes, _ := io.ReadAll(r.Body)
			logs[r.URL.Path] = bodyBytes
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
		sdk.WithHandler(
			func(ctx context.Context, logger sdk.Logger, req sdk.Request) (sdk.Response, error) {
				logger.Info("Request received", "request", req)
				panic(req)
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

	r := bytes.NewReader([]byte("{\"input1\": \"value1\"}"))

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
	req.Header.Set(sdk.ActivityFileUploadHeader, "/activity1/log")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Errorf("Error sending request: %v", err)
		return
	}

	if resp.StatusCode == http.StatusInternalServerError {
		if resp.Body != nil {
			defer resp.Body.Close()
			functionErr := new(sdk.ErrFunction)
			err := json.NewDecoder(resp.Body).Decode(&functionErr)
			if err != nil {
				t.Errorf("Error decoding error response: %v", err)
			}
		} else {
			t.Errorf("Unexpected response: %v", resp.Status)
		}
	}
	if resp.StatusCode == http.StatusOK {
		if resp.Body != nil {
			defer resp.Body.Close()
			var result sdk.Response
			err := json.NewDecoder(resp.Body).Decode(&result)
			if err != nil {
				t.Errorf("Error decoding response: %v", err)
			}
		} else {
			t.Errorf("Unexpected response: %v", resp.Status)
		}
	}

}
