package builder

import (
	"context"
	"io"
)

type goBuildContextGetter struct {
}

var _ BuildContextGetter = (*goBuildContextGetter)(nil)

func (bcg *goBuildContextGetter) GetBuildContext(context.Context, BuildContextGetterOptions, io.Writer) error {
	return nil
}
