// Adapted from https://github.com/openfaas/templates-sdk/blob/master/go-http/handler.go
// Original license: MIT
package sdk

import (
	"context"
	"log/slog"
)

type Logger interface {
	Log(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr)
}

type Request = map[string]any
type Response = map[string]any

// FunctionHandler used for a serverless Go method invocation
type FunctionHandler interface {
	Handle(ctx context.Context, logger Logger, req Request) (Response, error)
}
