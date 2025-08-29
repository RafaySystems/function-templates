import requests, json, time
from typing import Any, Callable, Optional

class StateClient:
    def __init__(self, base_url: str, token: str, org_id: str, 
                 project_id: Optional[str] = None, env_id: Optional[str] = None, timeout=5):
        self.base_url = base_url.rstrip("/")
        self.token = token
        self.org_id = org_id
        self.project_id = project_id
        self.env_id = env_id
        self.timeout = timeout

    # ---------- Scope helpers ----------
    @classmethod
    def for_org(cls, base_url: str, token: str, org_id: str):
        return cls(base_url, token, org_id)

    @classmethod
    def for_project(cls, base_url: str, token: str, org_id: str, project_id: str):
        return cls(base_url, token, org_id, project_id)

    @classmethod
    def for_env(cls, base_url: str, token: str, org_id: str, project_id: str, env_id: str):
        return cls(base_url, token, org_id, project_id, env_id)

    # ---------- Helpers ----------
    def _headers(self):
        headers = {
            "X-Eaas-State-Token": self.token,
            "X-Organization-ID": self.org_id,
        }
        if self.project_id:
            headers["X-Project-ID"] = self.project_id
        if self.env_id:
            headers["X-Environment-ID"] = self.env_id
        return headers

    def _get_raw(self, key: str):
        params = {"key": key, "organization_id": self.org_id}
        if self.project_id: params["project_id"] = self.project_id
        if self.env_id: params["environment_id"] = self.env_id

        resp = requests.get(self.base_url, headers=self._headers(),
                            params=params, timeout=self.timeout)
        if resp.status_code == 404:
            return None, ""
        resp.raise_for_status()
        data = resp.json()
        return data["value"], data.get("version", "")

    # ---------- Public API ----------
    def Get(self, key: str) -> Any:
        raw, _ = self._get_raw(key)
        return raw

    def Set(self, key: str, update_fn: Callable[[Any], Any], max_retries: int = 5) -> None:
        """Create/update with OCC retry. update_fn takes old_value -> new_value"""
        for attempt in range(max_retries):
            old_value, version = self._get_raw(key)
            new_value = update_fn(old_value)

            body = {
                "scope": {
                    "organization_id": self.org_id,
                    "project_id": self.project_id,
                    "environment_id": self.env_id
                },
                "key": key,
                "value": new_value,
                "version": version,
            }
            resp = requests.put(self.base_url,
                                 headers=self._headers(),
                                 json=body, timeout=self.timeout)
            if resp.status_code == 409:
                # OCC conflict, retry
                continue
            resp.raise_for_status()
            return
        raise Exception(f"Set failed after {max_retries} retries due to OCC conflicts")

    def Delete(self, key: str) -> None:
        resp = requests.delete(self.base_url,
                               headers=self._headers(),
                               params={"key": key, "organization_id": self.org_id},
                               timeout=self.timeout)
        if resp.status_code != 200 and resp.status_code != 404:
            resp.raise_for_status()