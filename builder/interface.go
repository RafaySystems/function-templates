package builder

import (
	"context"
	"io"
)

// BuildContextGetterOptions is the options for getting build context for a function
type BuildContextGetterOptions struct {
	Language string
	Source   string
	Imports  []string
}

// BuildContextGetter is an interface for getting build context for a function
type BuildContextGetter interface {
	GetBuildContext(context.Context, BuildContextGetterOptions, io.Writer) error
}
