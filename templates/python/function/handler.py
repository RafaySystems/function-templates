from typing import Dict, Any
from logging import Logger
from python_sdk_rafay_workflow import sdk

def handle(logger: Logger,request: Dict[str, Any]) -> Dict[str, Any]:
    logger.info(f"received request, req: {request}")

    counter = 0
    if "previous" in request:
        logger.info(f"previous counter, prev: {request['previous']}")
        counter = request["previous"].get("counter", 0)

    resp = {
        "output": "Hello World",
        "request": request,
    }

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
