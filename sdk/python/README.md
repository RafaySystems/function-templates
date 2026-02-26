# Environment Manager Python Function SDK

SDK for building Python functions invoked by the Environment Manager workflow engine. It handles HTTP serving (FastAPI), request metadata, and event context so you can focus on your function logic.

## Installation

```bash
pip install python_sdk_rafay_workflow
```

Or add the package to your project (e.g. from a local or private repo). For a working example, see [examples/python/cloudformation](../../examples/python/cloudformation).

## Quick start

Implement a handler and run the SDK:

```python
import logging
from python_sdk_rafay_workflow import serve_function

def handle(logger: logging.Logger, request: dict) -> dict:
    logger.info("request received")
    return {"result": "ok"}

if __name__ == "__main__":
    serve_function(handle)
```

## Handler and request/response

Your handler has the signature:

```python
def handle(logger: logging.Logger, request: dict) -> dict:
    ...
```

Handlers can be sync or async (`async def handle(logger, request): ...`).

- **request** is a dict containing the JSON body plus a **metadata** key filled from incoming headers: activity ID, environment ID/name, organization ID, project ID, state store URL/token, and **event source**, **event source name**, and **event type**. This metadata drives [EventDetails](#eventdetails) below.
- **response** is a dict returned as JSON under a `data` key in the HTTP response.

## EventDetails

Each invocation can carry event metadata (source, source name, and event type). **EventDetails** is the typed view of that metadata so you can branch or read names without parsing headers yourself.

### Obtaining EventDetails

```python
from python_sdk_rafay_workflow import EventDetails

event = EventDetails(request)
```

`EventDetails(request)` takes the handler’s request dict and populates `source`, `source_name`, and `type` from `request["metadata"]`.

### Fields

| Field         | Type  | Description                                |
|---------------|-------|--------------------------------------------|
| `source`      | str   | Event source (e.g. workload, action).      |
| `source_name` | str   | Name of the source resource.               |
| `type`        | str   | Event type (deploy, destroy, force-destroy). |

### Event types

Use the **EventType** enum for comparisons and string conversion:

| Member                | String value     |
|-----------------------|------------------|
| `EventType.DEPLOY`    | `"deploy"`       |
| `EventType.DESTROY`   | `"destroy"`      |
| `EventType.FORCE_DESTROY` | `"force-destroy"` |

Use `str(EventType.DEPLOY)` or `EventType.DEPLOY.value` to get the string.

### Sources

The engine may send events from these sources (used by the helpers below): `"action"`, `"schedules"`, `"workload"`, `"environment"`.

### Convenience helpers

**Source checks:** return True when the event’s source matches.

| Method           | Source      |
|------------------|-------------|
| `is_action()`    | action      |
| `is_schedules()` | schedules   |
| `is_workload()`  | workload    |
| (environment)    | use combined helpers below |

**Name getters:** return `(str, bool)` — the source name and True only when the event’s source matches.

| Method                 | Source      |
|------------------------|-------------|
| `get_action_name()`    | action      |
| `get_schedules_name()` | schedules   |
| `get_workload_name()`  | workload    |
| (environment)          | use combined helpers below |

**Type checks:** `is_deploy()`, `is_destroy()`, `is_force_destroy()`, `get_type_as_string()`.

**Combined (source + type):**

| Method                   | Condition                                          |
|--------------------------|----------------------------------------------------|
| `is_workload_deploy()`   | source is workload and type deploy                 |
| `is_workload_destroy()`  | source is workload and type destroy or force-destroy |
| `is_environment_deploy()`   | source is environment and type deploy          |
| `is_environment_destroy()`   | source is environment and type destroy or force-destroy |

### Example usage

```python
from python_sdk_rafay_workflow import EventDetails, EventType

def handle(logger, request):
    event = EventDetails(request)
    # create a variable called action with default value "deploy"
    action = str(EventType.DEPLOY)
    # if action is the source of this event, then get the action name
    name, ok = event.get_action_name()
    if ok:
        action = name
    if event.is_workload_deploy():
        name, _ = event.get_workload_name()
        # deploy workload "name"
    if event.is_environment_destroy():
        # teardown environment
    return {"ok": True}
```

## Errors

Raise the SDK exception types from your handler; the SDK encodes them as a structured JSON response with an error code so the engine can retry or handle appropriately:

| Exception                  | Use when |
|----------------------------|----------|
| `FailedException(message)` | Permanent failure; do not retry. |
| `TransientException(message)` | Temporary failure; engine may retry. |
| `ExecuteAgainException(message, **data)` | Ask engine to re-invoke (e.g. with updated data). |

Example:

```python
from python_sdk_rafay_workflow import FailedException

def handle(logger, request):
    if not request.get("required_field"):
        raise FailedException("required_field is required")
    return {"result": "ok"}
```

## Configuration

Call `serve_function` with optional arguments:

| Argument   | Default     | Description                |
|------------|-------------|----------------------------|
| `handler`  | (required)  | Your function handler.     |
| `host`     | `'0.0.0.0'` | Bind address.              |
| `port`     | `5000`      | HTTP port.                 |

Additional behavior is controlled via environment variables (e.g. `GUNICORN_WORKERS`, `LOG_FLUSH_TIMEOUT`, `skip_tls_verify`). See the package source for the full list.

## Example

The [examples/python/cloudformation](../../examples/python/cloudformation) directory contains a CloudFormation-based function that uses the SDK: handler signature, request parsing, and `FailedException`. You can use `EventDetails(request)` there to branch on event source and type.
