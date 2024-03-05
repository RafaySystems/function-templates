package function

import (
	"context"
	"fmt"

	sdk "github.com/RafaySystems/function-templates/sdk/go"
)

// Handle a function invocation
func Handle(ctx context.Context, logger sdk.Logger, req sdk.Request) (sdk.Response, error) {
	resp := sdk.Response{
		"message": fmt.Sprintf("Hello, Go. You said: %s", req["name"]),
	}

	return resp, nil
}
