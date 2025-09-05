from typing import Dict, Any
from logging import Logger
from python_sdk_rafay_workflow import sdk
from python_sdk_rafay_workflow.state_client import StateClientBuilder
import time

def handle(logger: Logger,request: Dict[str, Any]) -> Dict[str, Any]:
    logger.info(f"received request, req: {request}")

    counter = 0
    if "previous" in request:
        logger.info(f"previous counter, prev: {request['previous']}")
        counter = request["previous"].get("counter", 0)

    # Build state client for organization scope
    ostate = StateClientBuilder(request).with_org_scope()

   # Increment counter with OCC safe update
    ostate.Set("org_counter", lambda old: (old or 0) + 1)

    # Build state client for project scope
    pstate = StateClientBuilder(request).with_project_scope()

    # Increment counter without OCC safe update
    value, version = pstate.Get("project_counter")
    logger.info(f"project counter read: {value}, version: {version}")
    pstate.SetKV("project_counter", str(int(value or 0) + 1), version)

    # Build state client for environment scope
    state = StateClientBuilder(request).with_env_scope()

    # Increment counter with OCC safe update
    state.Set("env_counter", lambda old: (old or 0) + 1)

    value, version = state.Get("env_counter")
    logger.info(f"environment counter updated: {value}")

    if version > 2:
        # Simulate for testing delete
        state.Delete("env_counter")
        logger.info(f"environment counter deleted")

    resp = {
        "output": "Hello World",
        "request": request,
    }

    for i in range(request.get("count", 1)):
        logger.info(f"log iteration {i}")
        time.sleep(request.get("sleep", 1))

    if "error" in request:
        err_str = str(request["error"])
        if err_str == "execute_again":
            if counter > 1:
                return resp
            raise sdk.ExecuteAgainException(err_str, rkey="rvalue", counter=counter+1)
        elif err_str == "transient":
            raise sdk.TransientException(err_str)
        elif err_str == "failed":
            raise sdk.FailedException(err_str)
        else:
            raise Exception(err_str)
    return resp
