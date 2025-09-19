import asyncio
from io import BytesIO
from logging.handlers import MemoryHandler

import httpx

from .const import WorkflowTokenHeader


class ActivityLogHandler(MemoryHandler):
    def __init__(self, endpoint, token, timeout=10, verify=True, max_retries=3, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.endpoint = endpoint
        self.token = token
        self.timeout = timeout
        self.verify = verify
        self.max_retries = max_retries
        self._lock = asyncio.Lock()

        # Reuse one async client per handler to avoid recreating connections
        self.client = httpx.AsyncClient(timeout=httpx.Timeout(timeout), verify=verify)

    async def _send_logs(self, payload: bytes) -> httpx.Response:
        """Async log upload using httpx."""
        files = {"content": ("stdout", BytesIO(payload), "text/plain")}
        return await self.client.post(
            f"{self.endpoint}?append=true",
            headers={WorkflowTokenHeader: self.token},
            files=files,
        )

    def flush(self):
        """Synchronous flush that schedules async flush in background."""
        # Don't block - just schedule the async flush
        try:
            loop = asyncio.get_event_loop()
            if loop.is_running():
                # Schedule the async flush as a task
                asyncio.create_task(self.async_flush())
        except RuntimeError:
            # No event loop running, skip flush
            pass

    async def async_flush(self):
        """Fully async, non-blocking log flush."""
        async with self._lock:
            if not self.buffer:
                return

            buf = [self.format(record) for record in self.buffer]
            payload = ("\n".join(buf) + "\n").encode("utf-8")

            attempt = 0
            while attempt < self.max_retries:
                try:
                    resp = await self._send_logs(payload)
                    if resp.status_code == 200:
                        self.buffer.clear()
                        return
                    else:
                        attempt += 1
                        await asyncio.sleep(1)
                except Exception:
                    attempt += 1
                    await asyncio.sleep(1)

            # Only print error after max retries
            print(
                f"[ActivityLogHandler] Failed to send logs to {self.endpoint} "
                f"after {self.max_retries} attempts"
            )

    async def close(self):
        """Gracefully close the httpx client when shutting down."""
        await self.client.aclose()
