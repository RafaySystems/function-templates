"""Tests for StateClientBuilder SSL verification behaviour controlled by skip_tls_verify env var."""

import os
import unittest
from unittest.mock import MagicMock, patch

from python_sdk_rafay_workflow.state_client import StateClientBuilder


def make_request(state_url="http://example.com/state", state_token="token"):
    return {
        "metadata": {
            "stateStoreUrl": state_url,
            "stateStoreToken": state_token,
            "organizationID": "org-1",
            "projectID": "proj-1",
            "environmentID": "env-1",
        }
    }


class TestStateClientBuilderVerify(unittest.TestCase):
    """Tests that StateClientBuilder reads skip_tls_verify env var correctly."""

    def test_verify_true_by_default(self):
        """When skip_tls_verify is not set, verify should be True (SSL on)."""
        env = {k: v for k, v in os.environ.items() if k != "skip_tls_verify"}
        with patch.dict(os.environ, env, clear=True):
            builder = StateClientBuilder(make_request())
            self.assertTrue(builder.verify)

    def test_verify_false_when_skip_tls_verify_true(self):
        """When skip_tls_verify=true, verify should be False (SSL off)."""
        with patch.dict(os.environ, {"skip_tls_verify": "true"}):
            builder = StateClientBuilder(make_request())
            self.assertFalse(builder.verify)

    def test_verify_true_when_skip_tls_verify_false(self):
        """When skip_tls_verify=false, verify should be True (SSL on)."""
        with patch.dict(os.environ, {"skip_tls_verify": "false"}):
            builder = StateClientBuilder(make_request())
            self.assertTrue(builder.verify)

    def test_verify_case_insensitive(self):
        """skip_tls_verify=TRUE (uppercase) should also disable SSL."""
        with patch.dict(os.environ, {"skip_tls_verify": "TRUE"}):
            builder = StateClientBuilder(make_request())
            self.assertFalse(builder.verify)


class TestStateClientVerifyPropagation(unittest.TestCase):
    """Tests that verify is correctly passed from builder to StateClient for all scopes."""

    def test_org_scope_propagates_verify_false(self):
        with patch.dict(os.environ, {"skip_tls_verify": "true"}):
            client = StateClientBuilder(make_request()).with_org_scope()
            self.assertFalse(client.verify)

    def test_project_scope_propagates_verify_false(self):
        with patch.dict(os.environ, {"skip_tls_verify": "true"}):
            client = StateClientBuilder(make_request()).with_project_scope()
            self.assertFalse(client.verify)

    def test_env_scope_propagates_verify_false(self):
        with patch.dict(os.environ, {"skip_tls_verify": "true"}):
            client = StateClientBuilder(make_request()).with_env_scope()
            self.assertFalse(client.verify)

    def test_env_scope_propagates_verify_true_by_default(self):
        env = {k: v for k, v in os.environ.items() if k != "skip_tls_verify"}
        with patch.dict(os.environ, env, clear=True):
            client = StateClientBuilder(make_request()).with_env_scope()
            self.assertTrue(client.verify)


class TestStateClientVerifyPassedToRequests(unittest.TestCase):
    """Tests that verify=self.verify is actually passed to requests and httpx calls."""

    def _make_client(self, skip_tls: str):
        with patch.dict(os.environ, {"skip_tls_verify": skip_tls}):
            return StateClientBuilder(make_request()).with_env_scope()

    def test_get_raw_passes_verify_false(self):
        client = self._make_client("true")
        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_response.json.return_value = {"value": "hello", "version": 1}

        with patch("requests.get", return_value=mock_response) as mock_get:
            client._get_raw("my_key")
            _, kwargs = mock_get.call_args
            self.assertFalse(kwargs["verify"])

    def test_get_raw_passes_verify_true(self):
        client = self._make_client("false")
        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_response.json.return_value = {"value": "hello", "version": 1}

        with patch("requests.get", return_value=mock_response) as mock_get:
            client._get_raw("my_key")
            _, kwargs = mock_get.call_args
            self.assertTrue(kwargs["verify"])

    def test_set_kv_passes_verify_false(self):
        client = self._make_client("true")
        mock_response = MagicMock()
        mock_response.status_code = 200

        with patch("requests.put", return_value=mock_response) as mock_put:
            client.set_kv("my_key", "my_value", 1)
            _, kwargs = mock_put.call_args
            self.assertFalse(kwargs["verify"])

    def test_delete_passes_verify_false(self):
        client = self._make_client("true")
        mock_response = MagicMock()
        mock_response.status_code = 200

        with patch("requests.delete", return_value=mock_response) as mock_delete:
            client.delete("my_key")
            _, kwargs = mock_delete.call_args
            self.assertFalse(kwargs["verify"])


if __name__ == "__main__":
    unittest.main()
