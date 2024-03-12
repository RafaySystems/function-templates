package builder

import (
	"context"
	"io"
)

type goBuildContextGetter struct {
}

var _ BuildContextGetter = (*goBuildContextGetter)(nil)

/*
How to prepare the build context for a Go function
1. create temp directory
2. copy the function template from fixturesfs templates/{func_language} to the temp directory
3. render the templates using the function data
5. tar the temp directory
6. write the tar to the writer
*/

func (bcg *goBuildContextGetter) GetBuildContext(context.Context, BuildContextGetterOptions, io.Writer) error {
	return nil
}
