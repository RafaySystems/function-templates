import pytest_httpserver as httpserver
from fastapi.testclient import TestClient
import socket
import unittest
from contextlib import closing

import backoff
import pytest_httpserver as httpserver
from fastapi.testclient import TestClient
from python_sdk_rafay_workflow import const as sdk_const
from python_sdk_rafay_workflow import sdk

from .jira import handle, approve_issue


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
        self.data = {}
        
        
    def setUp(self) -> None:
        self.activity_api.start()
        # self.function_server.start()

    def tearDown(self) -> None:
        self.activity_api.stop()
        # self.function_server.terminate()
    
    def test_jira(self):
        self.activity_api.expect_request("/jira-func").respond_with_data("")
        resp = self.call_function()
        self.assertEqual(resp.json(), {"data":{"status": "Approved"}})

    @staticmethod
    def _retry(resp):
        if resp.status_code == 500:
            # approve after 2 retries
            resp_json = resp.json()
            if 'data' in resp_json and resp_json['data'].get('counter', 0) == 2:
                approve_issue(resp_json['data'].get('ticket_id'))
            return resp_json["error_code"] != sdk.ERROR_CODE_FAILED

    @backoff.on_predicate(backoff.expo, _retry, max_tries=5)
    def call_function(self):
        resp = self.client.post(self.function_url, json=self.data, headers={
            sdk_const.EngineAPIEndpointHeader: self.activity_api.url_for("/"),
            sdk_const.ActivityFileUploadHeader: "jira-func",
            sdk_const.WorkflowTokenHeader: "token",
            sdk_const.ActivityIDHeader: "activityID",
            sdk_const.EnvironmentIDHeader: "environmentID",
            })
        self.data["previous"] = resp.json().get("data", {})
        return resp
    
if __name__ == "__main__":
    unittest.main()