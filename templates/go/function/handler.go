package function

import (
	"context"
	"fmt"
	"net/http"

	sdk "github.com/RafaySystems/function-templates/sdk/go"
)

// Handle a function invocation
func Handle(ctx context.Context, req sdk.Request) (sdk.Response, error) {
	var err error

	message := fmt.Sprintf("Body: %s", string(req.Body))

	return sdk.Response{
		Body:       []byte(message),
		StatusCode: http.StatusOK,
	}, err
}
