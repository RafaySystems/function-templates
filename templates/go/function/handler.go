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

	// save state store at organization scope
	ostate := stateclient.NewBoundState(req).WithOrgScope()

	raw, version, err := ostate.Get(ctx, "response")
	if sdk.IsErrNotFound(err) {
		version = 1
	} else {
		version = version + 1
		oldResp := make(sdk.Response)
		if err := json.Unmarshal(raw, &oldResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal old response: %w", err)
		}
		oldResp["output"] = "Hello Universe"
		resp = oldResp
		logger.Info("loaded previous response from state store", "response", resp)
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}
	err = ostate.SetKV(ctx, "response", json.RawMessage(respBytes), version)
	if err != nil {
		return nil, fmt.Errorf("failed to set initial response in state: %w", err)
	}

	// save state store at project scope
	pstate := stateclient.NewBoundState(req).WithProjectScope()
	raw, version, err := pstate.Get(ctx, "project_payload")
	if sdk.IsErrNotFound(err) {
		err = pstate.SetKV(ctx, "project_payload", json.RawMessage(`{"counter": 1}`), version)
		if err != nil {
			return nil, fmt.Errorf("failed to set initial payload: %w", err)
		}
	} else {
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}
		if payload["counter"].(float64) == 1.0 {
			pstate.Delete(ctx, "project_payload")
		}
	}

	// Set the state with the incremented counter
	state := stateclient.NewBoundState(req).WithEnvScope()

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
