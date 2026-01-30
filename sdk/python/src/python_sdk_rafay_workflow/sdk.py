import multiprocessing
import os

import uvicorn

from .app_factory import create_app

# keep these imports since most of the handler.py are using them as `from python_sdk_rafay_workflow import sdk`,
# instead of `import python_sdk_rafay_workflow as sdk`
from .const import *
from .errors import *


def serve_function(handler, host='0.0.0.0', port=5000):
    app = create_app(handler)

    # For production/OpenFaaS environments, use gunicorn with uvicorn workers
    if os.environ.get("fprocess") == "python main.py" or os.environ.get("OPENFAAS") == "true":
        from gunicorn.app.base import BaseApplication

        class StandaloneApplication(BaseApplication):
            def __init__(self, application, options=None):
                self.options = options or {}
                self.application = application
                super().__init__()

            def load_config(self):
                for key, value in self.options.items():
                    self.cfg.set(key.lower(), value)

            def load(self):
                return self.application

        workers = int(os.environ.get("GUNICORN_WORKERS", "4"))
        threads = int(os.environ.get("GUNICORN_THREADS", "2"))
        print(f"Creating {workers} gunicorn workers with {threads} threads each")
        StandaloneApplication(app, {
            'bind': f'{host}:{port}',
            'workers': workers,
            'worker_class': 'uvicorn.workers.UvicornWorker',
            'threads': threads,
            'timeout': 0,
        }).run()
        return

    # For development, use uvicorn directly
    uvicorn.run(app, host=host, port=port)
