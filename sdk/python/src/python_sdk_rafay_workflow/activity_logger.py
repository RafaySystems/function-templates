import asyncio
from io import BytesIO
from logging.handlers import MemoryHandler

import httpx

from .const import WorkflowTokenHeader


class ActivityLogHandler(MemoryHandler):
    def __init__(
            self,
            endpoint: str,
            token: str,
            client: httpx.Client,
            verify: bool = True,
            max_retries: int = 3,
            interval: int = 10,
            *args,
            **kwargs
    ):
        super().__init__(*args, **kwargs)
        self.endpoint = endpoint
        self.token = token
        self.client = client
        self.verify = verify
        self.max_retries = max_retries
        self.interval = interval
        self._lock = asyncio.Lock()
        self._periodic_task = None
        self._shutdown_event = asyncio.Event()

        self._start_periodic_upload()

    async def _send_logs(self, payload: bytes) -> httpx.Response:
        """Async log upload using httpx."""
        return await self.client.post(
            f"{self.endpoint}?append=true",
            headers={WorkflowTokenHeader: self.token},
            files={"content": ("stdout", BytesIO(payload), "text/plain")},
        )

    def _start_periodic_upload(self):
        """Start the periodic upload task if an event loop is available."""
        try:
            loop = asyncio.get_event_loop()
            if loop.is_running():
                self._periodic_task = asyncio.create_task(self._periodic_upload_loop())
        except RuntimeError:
            # No event loop running, periodic uploads will be disabled
            pass

    async def _periodic_upload_loop(self):
        """Background task that uploads logs every interval second."""
        try:
            while not self._shutdown_event.is_set():
                try:
                    # Wait for either the interval or shutdown event
                    await asyncio.wait_for(
                        self._shutdown_event.wait(),
                        timeout=self.interval
                    )
                    # If we get here, shutdown was requested
                    break
                except asyncio.TimeoutError:
                    # Timeout means it's time for periodic upload
                    await self.async_flush()
        except asyncio.CancelledError:
            # Task was cancelled, exit gracefully
            pass

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

            for attempt in range(self.max_retries):
                try:
                    resp = await self._send_logs(payload)
                    if resp.status_code == 200:
                        self.buffer.clear()
                        return
                except Exception:
                    pass

                if attempt < self.max_retries - 1:
                    await asyncio.sleep(1)

            # Only print error after max retries
            print(
                f"[ActivityLogHandler] Failed to send logs to {self.endpoint} "
                f"after {self.max_retries} attempts"
            )

    async def close(self):
        """Gracefully close the handler, upload remaining logs, and clean up resources."""
        # Signal shutdown to stop periodic uploads
        self._shutdown_event.set()

        # Cancel and wait for a periodic task to complete
        if self._periodic_task and not self._periodic_task.done():
            self._periodic_task.cancel()
            try:
                await self._periodic_task
            except asyncio.CancelledError:
                pass

        # Final flush to ensure all remaining logs are uploaded
        # Uses a longer timeout for the final flush to ensure it completes
        try:
            await asyncio.wait_for(self.async_flush(), timeout=self.interval)
        except asyncio.TimeoutError:
            print(f"[ActivityLogHandler] Final flush timed out after {self.interval} seconds")
