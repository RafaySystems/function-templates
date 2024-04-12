from typing import *
import pytest_httpserver as httpserver
import unittest
import requests
from logging import Logger
from python_sdk_rafay_workflow import sdk
from python_sdk_rafay_workflow import const as sdk_const
from .jira import handle, approve_issue
import backoff

# from waitress import serve
import socket
from contextlib import closing
# import threading
# import multiprocessing

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
        self.data = {}
        # self.function_server = multiprocessing.Process(target=serve, args=(app,), kwargs={"host": "127.0.0.1", "port": port})
        
        
    def setUp(self) -> None:
        self.activity_api.start()
        # self.function_server.start()

    def tearDown(self) -> None:
        self.activity_api.stop()
        # self.function_server.terminate()
    
    def test_jira(self):
        self.activity_api.expect_request("/jira-func").respond_with_data("")
        resp = self.call_function()
        self.assertEqual(resp.json, {"data":{"status": "Approved"}})

    @staticmethod
    def _retry(resp):
        if resp.status_code == 500:
            # approve after 2 retries
            if 'data' in resp.json and resp.json['data'].get('counter', 0) == 2:
                approve_issue(resp.json['data'].get('ticket_id'))
            return resp.json["error_code"] != sdk.ERROR_CODE_FAILED

    @backoff.on_predicate(backoff.expo, _retry, max_tries=5)
    def call_function(self):
        resp = self.client.post(self.function_url, json=self.data, headers={
            sdk_const.EngineAPIEndpointHeader: self.activity_api.url_for("/"),
            sdk_const.ActivityFileUploadHeader: "jira-func",
            sdk_const.WorkflowTokenHeader: "token",
            sdk_const.ActivityIDHeader: "activityID",
            sdk_const.EnvironmentIDHeader: "environmentID",
            })
        self.data["previous"] = resp.json.get("data", {})
        return resp
    
if __name__ == "__main__":
    unittest.main()