from typing import Dict, Any, Tuple
from flask import Flask, request, jsonify
from waitress import serve
import logging
import sys
import os
import json

from .activity_logger import ActivityLogHandler
from .const import *
from .errors import *

FUNCTION_NAME=os.environ.get('FUNCTION_NAME', 'default-function-name')
LOG_LEVEL=os.environ.get('LOG_LEVEL', 'INFO')
LOG_BUFFER_CAPACITY=int(os.environ.get('LOG_BUFFER_CAPACITY', "10"))

_format = "time=%(asctime)s level=%(levelname)s path=%(pathname)s line=%(lineno)d msg=%(message)s"
_logger = logging.Logger(FUNCTION_NAME)
_handler = logging.StreamHandler(stream=sys.stdout)
_formatter = logging.Formatter(_format)
_handler.setFormatter(_formatter)
_logger.addHandler(_handler)
_handler.setLevel(LOG_LEVEL)

def log(f):
    def wrap(*args, **kwargs):
        activity_id = request.headers.get(ActivityIDHeader, default="", type=str)
        environment_id = request.headers.get(EnvironmentIDHeader, default="", type=str)
        environment_name = request.headers.get(EnvironmentNameHeader, default="", type=str)
        engine_endpoint= request.headers.get(EngineAPIEndpointHeader, type=str)
        file_upload_path= request.headers.get(ActivityFileUploadHeader, type=str)

        logger = logging.Logger(activity_id)
        extra = {
            "activity_id": activity_id,
            "environment_id": environment_id,
            "environment_name": environment_name,
        }

        token = request.headers.get(WorkflowTokenHeader)

        endpoint = engine_endpoint + file_upload_path
        logging_handler = ActivityLogHandler(endpoint=endpoint, token=token, capacity=LOG_BUFFER_CAPACITY)
        logging_handler.setFormatter(logging.Formatter(_format))
        logger.setLevel(LOG_LEVEL)
        logger.addHandler(logging_handler)
        logger.info(f"invoking function: {FUNCTION_NAME}")

        local_logger = logging.LoggerAdapter(_logger, extra)
        local_logger.info(f"invoking function: activity_id={activity_id}, environment_id={environment_id}, environment_name={environment_name}")

        resp = f(logger=logging.LoggerAdapter(logger, extra), *args, **kwargs)
        logging_handler.close()
        return resp
    return wrap


def call_ready():
    return jsonify({ "status": "ready" }), 200

def call(handler):
    return lambda: handle(handler)

@log
def handle(handler, logger=None) -> Tuple[Dict[str, Any], int]:
    resp, status_code = None, 0
    try:
        req = json.loads(request.data)
        if req is None:
            req = {}
        req["metadata"] = {
            "activityID": request.headers.get(ActivityIDHeader),
            "environmentID": request.headers.get(EnvironmentIDHeader),
            "environmentName": request.headers.get(EnvironmentNameHeader),
        }
        resp = handler(logger, req)
        resp, status_code = jsonify({"data": resp}), 200
    except ExecuteAgainException as e:
        resp, status_code = jsonify(e.__dict__), 500
    except FailedException as e:
        resp, status_code = jsonify(e.__dict__), 500
    except TransientException as e:
        resp, status_code = jsonify(e.__dict__), 500
    except Exception as e:
        resp, status_code = jsonify(errorCode=ERROR_CODE_FAILED,message=str(e)), 500
    return resp, status_code

def _get_app(handler):
    app = Flask(FUNCTION_NAME)

    app.add_url_rule('/_/ready', methods=['GET'], view_func=call_ready)
    app.add_url_rule('/', methods=['POST'], view_func=call(handler))
    return app

def serve_function(handler, host='0.0.0.0', port=5000, ):
    _logger.info(f'Starting Python Function {FUNCTION_NAME} ...')

    app = Flask(FUNCTION_NAME)

    app.add_url_rule('/_/ready', methods=['GET'], view_func=call_ready)
    app.add_url_rule('/', methods=['POST'], view_func=call(handler))

    serve(app, host=host, port=port)
