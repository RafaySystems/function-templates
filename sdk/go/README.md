# Environment Manager Go Function SDK

SDK for building Go functions invoked by the Environment Manager workflow engine. It handles HTTP serving, request metadata, and event context so you can focus on your function logic.

## Installation

```bash
go get github.com/RafaySystems/function-templates/sdk/go
```

For a working example that uses the SDK, see [examples/go/helm/go.mod](../../examples/go/helm/go.mod). Local development can use a `replace` directive pointing at the SDK path.

## Quick start

Implement a handler and run the SDK:

```go
package main

import (
	"context"
	sdk "github.com/RafaySystems/function-templates/sdk/go"
)

func Handle(ctx context.Context, logger sdk.Logger, req sdk.Request) (sdk.Response, error) {
	logger.Info("request received")
	return sdk.Response{"result": "ok"}, nil
}

func main() {
	f, err := sdk.NewFunctionSDK(sdk.WithHandler(Handle))
	if err != nil {
		panic(err)
	}
	_ = f.Run(context.Background())
}
```

## Handler and request/response

Your handler has the signature:

```go
func(ctx context.Context, logger sdk.Logger, req sdk.Request) (sdk.Response, error)
```

- **Request** and **Response** are map-like types (`map[string]any`). Use `req["key"]` or helpers such as `req.GetString("key")` for typed access. Nested keys are supported (e.g. `req.GetString("nested", "field")`).
- Request **metadata** is filled from incoming headers: activity ID, environment ID/name, organization ID, project ID, state store URL/token, and **event source**, **event source name**, and **event type**. This metadata drives [EventDetails](#eventdetails) below.

## EventDetails

Each invocation can carry event metadata (source, source name, and event type). **EventDetails** is the typed view of that metadata so you can branch or read names without parsing headers yourself.

### Obtaining EventDetails

```go
event := sdk.NewEventDetails(req)
```

`NewEventDetails` takes the handler’s `Request` and returns `*EventDetails` with `Source`, `SourceName`, and `Type` populated from request metadata.

### Fields

| Field       | Type        | Description                          									|
|-------------|-------------|-----------------------------------------------------------------------|
| `Source`    | `string`    | Event source (e.g. workload, action).									|
| `SourceName`| `string`    | Name of the source resource (e.g action name, workload name).         |
| `Type`      | `EventType` | Kind of event (deploy, destroy, etc.).								|

### Event types

| Constant              | String value     |
|-----------------------|------------------|
| `sdk.DeployEventType` | `"deploy"`       |
| `sdk.DestroyEventType`| `"destroy"`      |
| `sdk.ForceDestroyEventType` | `"force-destroy"` |

### Sources

The engine may send events from these sources (used by the helpers below): `"action"`, `"schedules"`, `"workload"`, `"environment"`.

### Convenience helpers

**Source checks:** return true when the event’s source matches.

| Method        | Source      |
|---------------|-------------|
| `IsAction()`  | action      |
| `IsSchedules()` | schedules |
| `IsWorkload()`  | workload  |
| (environment) | use combined helpers below |

**Name getters:** return `(string, bool)` — the source name and true only when the event’s source matches.

| Method               | Source      |
|----------------------|-------------|
| `GetActionName()`    | action      |
| `GetSchedulesName()` | schedules   |
| `GetWorkloadName()`  | workload    |
| (environment)        | use combined helpers below |

**Type checks:** `IsDeploy()`, `IsDestroy()`, `IsForceDestroy()`, `GetTypeAsString()`.

**Combined (source + type):**

| Method                 | Condition                          |
|------------------------|------------------------------------|
| `IsWorkloadDeploy()`   | source is workload and type deploy |
| `IsWorkloadDestroy()`  | source is workload and type destroy or force-destroy |
| `IsEnvironmentDeploy()`| source is environment and type deploy |
| `IsEnvironmentDestroy()` | source is environment and type destroy or force-destroy |

### Example usage

```go
func Handle(ctx context.Context, logger sdk.Logger, req sdk.Request) (sdk.Response, error) {
	event := sdk.NewEventDetails(req)
	// create a variable called action with default value `deploy`
	action := string(sdk.DeployEventType)
	// if action is the source of this event, then get the action name
	if name, ok := event.GetActionName(); ok {
		action = name
	}
	if event.IsWorkloadDeploy() {
		name, _ := event.GetWorkloadName()
		// deploy workload "name"
	}
	if event.IsEnvironmentDestroy() {
		// teardown environment
	}
	return sdk.Response{}, nil
}
```

## Errors

Return an error from your handler; the SDK encodes it as a structured JSON response with an error code. Use the typed constructors so the engine can retry or handle appropriately:

| Constructor              | Use when |
|--------------------------|----------|
| `sdk.NewErrFailed(msg)`  | Permanent failure; do not retry. |
| `sdk.NewErrTransient(msg)` | Temporary failure; engine may retry. |
| `sdk.NewErrExecuteAgain(msg, data)` | Ask engine to re-invoke (e.g. with updated data). |
| `sdk.NewErrNotFound(msg)` | Resource not found. |
| `sdk.NewErrConflict(msg)` | Conflict (e.g. version mismatch). |

The response shape is `ErrFunction` with `ErrCode` (e.g. `ErrCodeFailed`, `ErrCodeTransient`, `ErrCodeExecuteAgain`). The engine may retry on transient or execute-again errors.

## Configuration

Pass options to `NewFunctionSDK`:

| Option | Description |
|--------|-------------|
| `WithHandler(handler)` | Required. Your function handler. |
| `WithPort(port)` | HTTP port (default if not set). |
| `WithReadTimeout`, `WithWriteTimeout` | HTTP timeouts. |
| `WithShutdownTimeout` | Graceful shutdown timeout. |
| `WithLogLevel(level)` | Log level (e.g. `slog.LevelDebug`). |
| `WithListener(listener)` | Custom listener instead of default bind. |
| `WithLogWriteTimeout`, `WithLogFlushRate`, `WithLogUploadRetryCount` | Log upload behavior. |
| `WithServerSkipTLSVerify(bool)` | Skip TLS verification for log upload. |

See [sdk.go](sdk.go) for the full list of `With*` options.

## Example

The [examples/go/helm](../../examples/go/helm) directory contains a Helm-based function that uses the SDK: handler signature, request parsing, and typed errors. You can use `sdk.NewEventDetails(req)` there to branch on event source and type.
