import socket
import unittest
from contextlib import closing
from logging import Logger
from typing import *

import pytest_httpserver as httpserver
from fastapi.testclient import TestClient
from python_sdk_rafay_workflow import const as sdk_const
from python_sdk_rafay_workflow import sdk


def handle(logger: Logger, request: Dict[str, Any]) -> Dict[str, Any]:
    logger.info("inside function handler1, activityID: %s" % (request["metadata"]["activityID"]))
    logger.info("inside function handler2, activityID: %s" % (request["metadata"]["activityID"]))
    for key in range(10):
        logger.info("inside function handler %s, activityID: %s" % (key, request["metadata"]["activityID"]))
    return {
        "message": "Hello, World!"
    }


def find_free_port():
    with closing(socket.socket(socket.AF_INET, socket.SOCK_STREAM)) as s:
        s.bind(('', 0))
        s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        return s.getsockname()[1]


class TestSDK(unittest.TestCase):

    def __init__(self, *args, **kwargs):
        super(TestSDK, self).__init__(*args, **kwargs)
        self.activity_api = httpserver.HTTPServer()
        app = sdk._get_app(handle)
        self.client = TestClient(app)
        self.function_url = "/"
        # self.function_server = multiprocessing.Process(target=serve, args=(app,), kwargs={"host": "127.0.0.1", "port": port})

    def setUp(self) -> None:
        self.activity_api.start()
        # self.function_server.start()

    def tearDown(self) -> None:
        self.activity_api.stop()
        # self.function_server.terminate()

    def test_sdk(self):
        self.activity_api.expect_request("/foobar").respond_with_handler(my_custom_handler)
        resp = self.client.post(self.function_url, json={"foo": "bar"}, headers={
            sdk_const.EngineAPIEndpointHeader: self.activity_api.url_for("/"),
            sdk_const.ActivityFileUploadHeader: "foobar",
            sdk_const.WorkflowTokenHeader: "token",
            sdk_const.ActivityIDHeader: "activityID",
            sdk_const.EnvironmentIDHeader: "environmentID",
        })
        self.assertEqual(resp.json(), {"data": {"message": "Hello, World!"}})


def my_custom_handler(request):
    if request.method == 'POST':
        if request.headers.get('Content-Type') == 'application/json':
            data = request.get_json()
            response_data = {"received": data}
            return 200, {"Content-Type": "application/json"}, response_data
        else:
            return 415, {"Content-Type": "text/plain"}, "Unsupported Media Type"
    else:
        return 405, {"Content-Type": "text/plain"}, "Method Not Allowed"


if __name__ == "__main__":
    unittest.main()
