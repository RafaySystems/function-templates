package function

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sdk "github.com/RafaySystems/function-templates/sdk/go"
	stateclient "github.com/RafaySystems/function-templates/sdk/go/pkg/state"
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

	// Set the state with the incremented counter
	state := stateclient.NewBoundState(req).WithEnvScope()

	//Increment counter and set it to store
	err := state.SetKV(ctx, "counter", json.RawMessage(fmt.Sprintf("%f", counter)))
	if err != nil {
		return nil, err
	}
	resp["counter"] = counter

	// Set interim payload safely and set it to store
	err = state.Set(ctx, "payload", func(old json.RawMessage) (json.RawMessage, error) {
		if len(old) == 0 {
			return json.Marshal(map[string]any{
				"counter": 1.0,
			})
		}
		var oldValue map[string]any
		if err := json.Unmarshal(old, &oldValue); err != nil {
			return nil, fmt.Errorf("failed to unmarshal old value: %w", err)
		}
		newValue := oldValue["counter"].(float64) + 1
		if new, err := json.Marshal(map[string]any{
			"counter": newValue,
		}); err != nil {
			return nil, fmt.Errorf("failed to marshal new value: %w", err)
		} else {
			resp["payload"] = newValue
			logger.Info("incremented counter within payload", "new_value", newValue)
			return new, nil
		}
	})
	if err != nil {
		return nil, err
	}

	count, err := req.GetInt("count")
	if err != nil {
		return nil, sdk.NewErrFailed("count is not an integer")
	}

	for i := 0; i < count; i++ {
		logger.Info("log iteration", "number", i)
		time.Sleep(1 * time.Second)
	}

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
		case "failed":
			return nil, sdk.NewErrFailed(errString)
		default:
			return nil, fmt.Errorf("unknown error: %s", errString)
		}
	}

	return sdk.Response(resp), nil
}
