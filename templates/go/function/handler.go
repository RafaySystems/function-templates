package function

import (
	"context"

	sdk "github.com/RafaySystems/function-templates/sdk/go"
)

func Handle(ctx context.Context, logger sdk.Logger, req sdk.Request) (sdk.Response, error) {
	logger.Info("received request", "req", req)

	counter := 0.0
	if prev, ok := req["previous"]; ok {
		logger.Info("previous request", "prev", prev)
		counter = prev.(map[string]any)["counter"].(float64)
	}

	resp := make(sdk.Response)
	resp["output"] = "Hello World"
	resp["request"] = req

	if err, ok := req["error"]; ok {
		errString, _ := err.(string)
		switch errString {
		case "execute_again":
			if counter > 1 {
				break
			}
			return nil, sdk.NewErrExecuteAgain(errString, map[string]any{
				"rkey":    "rvalue",
				"counter": counter + 1,
			})
		case "transient":
			return nil, sdk.NewErrTransient(errString)
		default:
			return nil, sdk.NewErrFailed(errString)
		}
	}

	return sdk.Response(resp), nil
}
