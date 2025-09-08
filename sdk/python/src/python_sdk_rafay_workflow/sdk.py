import asyncio
import inspect
import json
import logging
import os
import sys
from typing import Dict, Any

import uvicorn
from fastapi import FastAPI, Request, Response
from fastapi.responses import JSONResponse

from .activity_logger import ActivityLogHandler
from .const import *
from .errors import *

FUNCTION_NAME = os.environ.get('FUNCTION_NAME', 'default-function-name')
LOG_LEVEL = os.environ.get('LOG_LEVEL', 'INFO')
LOG_BUFFER_CAPACITY = int(os.environ.get('LOG_BUFFER_CAPACITY', "10"))
LOG_FLUSH_TIMEOUT = int(os.environ.get('LOG_FLUSH_TIMEOUT', "10"))
SKIP_TLS_VERIFY = os.environ.get('skip_tls_verify', "false")

_format = "time=%(asctime)s level=%(levelname)s path=%(pathname)s line=%(lineno)d msg=%(message)s"
_logger = logging.Logger(FUNCTION_NAME)
_handler = logging.StreamHandler(stream=sys.stdout)
_formatter = logging.Formatter(_format)
_handler.setFormatter(_formatter)
_logger.addHandler(_handler)
_handler.setLevel(LOG_LEVEL)


def log(f):
    async def wrap(request: Request, *args, **kwargs):
        activity_id = request.headers.get(ActivityIDHeader, "")
        environment_id = request.headers.get(EnvironmentIDHeader, "")
        environment_name = request.headers.get(EnvironmentNameHeader, "")
        engine_endpoint = request.headers.get(EngineAPIEndpointHeader)
        file_upload_path = request.headers.get(ActivityFileUploadHeader)

        logger = logging.Logger(activity_id)
        extra = {
            "activity_id": activity_id,
            "environment_id": environment_id,
            "environment_name": environment_name,
        }

        token = request.headers.get(WorkflowTokenHeader)

        endpoint = engine_endpoint + file_upload_path
        logging_handler = ActivityLogHandler(endpoint=endpoint, token=token, capacity=LOG_BUFFER_CAPACITY,
                                             timeout=LOG_FLUSH_TIMEOUT, verify=(SKIP_TLS_VERIFY != "true"))
        logging_handler.setFormatter(logging.Formatter(_format))
        logger.setLevel(LOG_LEVEL)
        logger.addHandler(logging_handler)

        log_format = "time=%(asctime)s level=%(levelname)s path=%(pathname)s line=%(lineno)d environment_name=%(environment_name)s environment_id=%(environment_id)s activity_id=%(activity_id)s  msg=%(message)s"
        stdout_handler = logging.StreamHandler(stream=sys.stdout)
        stdout_handler.setFormatter(logging.Formatter(log_format))

        logger.addHandler(stdout_handler)
        logger.info(f"invoking function: {FUNCTION_NAME}", extra=extra)

        resp = await f(request=request, logger=logging.LoggerAdapter(logger, extra), *args, **kwargs)
        logging_handler.close()
        stdout_handler.close()
        return resp

    return wrap


async def call_ready():
    return {"status": "ready"}


def call(handler):
    @log
    async def wrapped_handler(request: Request, logger=None):
        return await handle(handler, request, logger)

    return wrapped_handler


async def run_handler(handler, logger, req):
    if inspect.iscoroutinefunction(handler):
        # if the user handler is async — await directly
        return await handler(logger, req)
    else:
        # if the user handler is sync — run in a thread pool
        return await asyncio.to_thread(handler, logger, req)


async def handle(handler, request: Request, logger=None) -> Dict[str, Any] | Response:
    try:
        body = await request.body()
        req = json.loads(body) if body else {}
        req["metadata"] = {
            "activityID": request.headers.get(ActivityIDHeader),
            "environmentID": request.headers.get(EnvironmentIDHeader),
            "environmentName": request.headers.get(EnvironmentNameHeader),
        }
        resp = await run_handler(handler, logger, req)
        return {"data": resp}
    except ExecuteAgainException as e:
        return JSONResponse(e.__dict__, 500)
    except FailedException as e:
        return JSONResponse(e.__dict__, 500)
    except TransientException as e:
        return JSONResponse(e.__dict__, 500)
    except Exception as e:
        return JSONResponse(content={"error_code": ERROR_CODE_FAILED, "message": str(e)}, status_code=500)


def _get_app(handler):
    app = FastAPI(title=FUNCTION_NAME)

    wrapped_handler = call(handler)

    @app.get('/_/ready')
    async def ready():
        return await call_ready()

    @app.post('/')
    async def main(request: Request):
        return await wrapped_handler(request)

    return app


def serve_function(handler, host='0.0.0.0', port=5000):
    _logger.info(f'Starting Python Function {FUNCTION_NAME}')

    app = _get_app(handler)

    uvicorn.run(app, host=host, port=port)
