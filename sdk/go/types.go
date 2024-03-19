// Adapted from https://github.com/openfaas/templates-sdk/blob/master/go-http/handler.go
// Original license: MIT
package sdk

import (
	"context"
	"log/slog"
)

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Log(ctx context.Context, level slog.Level, msg string, args ...any)
}

type Object map[string]any

type Request = Object
type Response = Object

func (r Object) GetAsString(key string) (string, bool) {
	val, ok := r[key].(string)
	return val, ok
}

// FunctionHandler used for a serverless Go method invocation
type FunctionHandler interface {
	Handle(ctx context.Context, logger Logger, req Request) (Response, error)
}

type Handler func(ctx context.Context, logger Logger, req Request) (Response, error)

const (
	ActivityIDHeader         = "X-Activity-ID"
	EnvironmentIDHeader      = "X-Environment-ID"
	EnvironmentNameHeader    = "X-Environment-Name"
	WorkflowTokenHeader      = "X-Workflow-Token"
	EngineAPIEndpointHeader  = "X-Engine-Endpoint"
	ActivityFileUploadHeader = "X-Activity-File-Upload"
)

type ReadyResponse struct {
	Ready          bool  `json:"ready"`
	NumConnections int32 `json:"num_connections"`
}
