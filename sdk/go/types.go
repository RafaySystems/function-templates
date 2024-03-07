// Adapted from https://github.com/openfaas/templates-sdk/blob/master/go-http/handler.go
// Original license: MIT
package sdk

import (
	"context"
	"log/slog"
)

type Logger interface {
	Log(ctx context.Context, level slog.Level, msg string, args ...any)
}

type Request = map[string]any
type Response = map[string]any

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
