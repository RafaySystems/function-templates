from typing import *
import pytest_httpserver as httpserver
import unittest
import requests
from logging import Logger
from python_sdk_rafay_workflow import sdk
from python_sdk_rafay_workflow import const as sdk_const

from waitress import serve
import socket
from contextlib import closing
import threading
import multiprocessing

def handle(logger: Logger,request: Dict[str, Any]) -> Dict[str, Any]:
    logger.info("inside function handler1, activityID: %s" % (request["metadata"]["activityID"]))
    logger.info("inside function handler2, activityID: %s" % (request["metadata"]["activityID"]))
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
        port = find_free_port()
        app.testing = True
        self.client = app.test_client()
        self.function_url = "/"
        # self.function_server = multiprocessing.Process(target=serve, args=(app,), kwargs={"host": "127.0.0.1", "port": port})
        
        
    def setUp(self) -> None:
        self.activity_api.start()
        # self.function_server.start()

    def tearDown(self) -> None:
        self.activity_api.stop()
        # self.function_server.terminate()


    def test_sdk(self):
        self.activity_api.expect_request("/foobar").respond_with_data("")
        resp = self.client.post(self.function_url, json={"foo": "bar"}, headers={
            sdk_const.EngineAPIEndpointHeader: self.activity_api.url_for("/"),
            sdk_const.ActivityFileUploadHeader: "foobar",
            sdk_const.WorkflowTokenHeader: "token",
            sdk_const.ActivityIDHeader: "activityID",
            sdk_const.EnvironmentIDHeader: "environmentID",
            })
        self.assertEqual(resp.json, {"data":{"message": "Hello, World!"}})
        
if __name__ == "__main__":
    unittest.main()
