from logging.handlers import MemoryHandler
import requests
from .const import WorkflowTokenHeader
from io import BytesIO


class ActivityLogHandler(MemoryHandler):
    def __init__(self, endpoint, token, *args, **kwargs):
        MemoryHandler.__init__(self, *args, **kwargs)
        self.token = token
        self.endpoint = endpoint
        self.timeout = kwargs.get('timeout', 5)

    def flush(self) -> None:
        self.acquire()
        try:
            if len(self.buffer) > 0:
                buf = [self.format(record) for record in self.buffer]
                joined_string = '\n'.join(buf) + '\n'
                part = BytesIO(joined_string.encode('utf-8'))
                files = {
                    'content': ('stdout', part, 'text/plain'),
                }
                resp = requests.post(f'{self.endpoint}?append=true', headers={
                    WorkflowTokenHeader: self.token,
                }, files=files, timeout=self.timeout)

                if resp.status_code != 200:
                    print(f"Failed to send logs to {self.endpoint}, status code: {resp.status_code}, response: {resp.text}")
                self.buffer.clear()
        except Exception as e:
            print(f"Failed to send logs to {self.endpoint}, exception: {e}")
        finally:
            self.release()
