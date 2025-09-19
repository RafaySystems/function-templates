import requests
from typing import Any, Callable, Optional

class StateScope:
    def __init__(self, organization_id, project_id=None, environment_id=None):
        self.organization_id = organization_id
        self.project_id = project_id
        self.environment_id = environment_id

class StateClientBuilder:
    def __init__(self, request):
        meta = request["metadata"]
        self.base_url = meta["stateStoreUrl"]
        self.token = meta["stateStoreToken"]
        self.organization_id = meta["organizationID"]
        self.project_id = meta.get("projectID")
        self.environment_id = meta.get("environmentID")

    def with_org_scope(self):
        return StateClient(self.base_url, self.token, self.organization_id, self.project_id, self.environment_id, StateScope(self.organization_id))

    def with_project_scope(self):
        return StateClient(self.base_url, self.token, self.organization_id, self.project_id, self.environment_id, StateScope(self.organization_id, self.project_id))

    def with_env_scope(self):
        return StateClient(self.base_url, self.token, self.organization_id, self.project_id, self.environment_id, StateScope(self.organization_id, self.project_id, self.environment_id))

class StateClient:
    def __init__(self, base_url, token, organization_id, project_id, environment_id, scope: StateScope, timeout=5):
        self.base_url = base_url.rstrip("/")
        self.token = token
        self.scope = scope
        self.timeout = timeout
        self.organization_id = organization_id
        self.project_id = project_id
        self.environment_id = environment_id

    # ---------- Helpers ----------
    def _headers(self):
        headers = {
            "X-Eaas-State-Token": self.token,
            "X-Organization-ID": self.organization_id,
            "X-Project-ID": self.project_id,
            "X-Environment-ID": self.environment_id,
            "Content-Type": "application/json",
        }
        return headers
    
    # ...implement methods using self.scope...

    # ---------- Async Helpers ----------
    async def _get_raw_async(self, key: str):
        import httpx
        params = {"key": key, "organization_id": self.organization_id}
        if self.scope.project_id:
            params["project_id"] = self.scope.project_id
        if self.scope.environment_id:
            params["environment_id"] = self.scope.environment_id
        async with httpx.AsyncClient(timeout=self.timeout) as client:
            resp = await client.get(self.base_url, headers=self._headers(), params=params)
            if resp.status_code == 404:
                return None, 0
            resp.raise_for_status()
            data = resp.json()
            return data["value"], int(data.get("version", 0))

    def _get_raw(self, key: str):
        params = {"key": key, "organization_id": self.organization_id}
        if self.scope.project_id: params["project_id"] = self.scope.project_id
        if self.scope.environment_id: params["environment_id"] = self.scope.environment_id

        resp = requests.get(self.base_url, headers=self._headers(),
                            params=params, timeout=self.timeout)
        if resp.status_code == 404:
            return None, 0
        resp.raise_for_status()
        data = resp.json()
        return data["value"], int(data.get("version", 0))

    # ---------- Public API ----------
    def get(self, key: str) -> Any:
        raw, version = self._get_raw(key)
        return raw, version

    async def get_async(self, key: str) -> Any:
        raw, version = await self._get_raw_async(key)
        return raw, version

    def set_kv(self, key: str, value: str, version: int) -> None:
        """Create/update without OCC retry and let consumer handle conflicts"""
        body = {
            "scope": self.scope.__dict__,
            "key": key,
            "value": value,
            "version": version
        }
        resp = requests.put(self.base_url,
                                headers=self._headers(),
                                json=body, timeout=self.timeout)

        resp.raise_for_status()
        return

    async def set_kv_async(self, key: str, value: str, version: int) -> None:
        import httpx
        body = {
            "scope": self.scope.__dict__,
            "key": key,
            "value": value,
            "version": version
        }
        async with httpx.AsyncClient(timeout=self.timeout) as client:
            resp = await client.put(self.base_url, headers=self._headers(), json=body)
            resp.raise_for_status()
        return

    def set(self, key: str, update_fn: Callable[[Any], Any], max_retries: int = 5) -> None:
        """Create/update with OCC retry. update_fn takes old_value -> new_value"""
        for attempt in range(max_retries):
            old_value, version = self._get_raw(key)
            new_value = update_fn(old_value)

            body = {
                "scope": self.scope.__dict__,
                "key": key,
                "value": new_value,
                "version": version
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

    async def set_async(self, key: str, update_fn: Callable[[Any], Any], max_retries: int = 5) -> None:
        import httpx
        for attempt in range(max_retries):
            old_value, version = await self._get_raw_async(key)
            new_value = update_fn(old_value)
            body = {
                "scope": self.scope.__dict__,
                "key": key,
                "value": new_value,
                "version": version
            }
            async with httpx.AsyncClient(timeout=self.timeout) as client:
                resp = await client.put(self.base_url, headers=self._headers(), json=body)
                if resp.status_code == 409:
                    continue
                resp.raise_for_status()
                return
        raise Exception(f"set_async failed after {max_retries} retries due to OCC conflicts")

    def delete(self, key: str) -> None:
        body = {
                "scope": self.scope.__dict__,
                "key": key
            }
        resp = requests.delete(self.base_url,
                               headers=self._headers(),
                               json=body, timeout=self.timeout)
        if resp.status_code != 200 and resp.status_code != 404:
            resp.raise_for_status()

    async def delete_async(self, key: str) -> None:
        import httpx
        body = {
            "scope": self.scope.__dict__,
            "key": key
        }
        async with httpx.AsyncClient(timeout=self.timeout) as client:
            resp = await client.request("DELETE", self.base_url, headers=self._headers(), json=body)
            if resp.status_code != 200 and resp.status_code != 404:
                resp.raise_for_status()