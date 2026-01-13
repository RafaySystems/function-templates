import asyncio
import inspect
import os
from concurrent.futures import ThreadPoolExecutor
from contextlib import asynccontextmanager

import httpx
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

from .const import *
from .errors import *
from .logger import log


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup phase - initialize resources
    app.state.loggers = set()
    app.state.executor = ThreadPoolExecutor(
        max_workers=int(os.environ.get("MAX_WORKERS", "50"))
    )

    # create a shared httpx client for all activity loggers
    app.state.httpx_client = httpx.AsyncClient(
        timeout=httpx.Timeout(int(os.environ.get("LOG_FLUSH_TIMEOUT", "10"))),
        verify=os.environ.get('skip_tls_verify', "false") != "true",
        limits=httpx.Limits(max_connections=int(os.environ.get("LOG_UPLOAD_MAX_CONNECTIONS", "100")))
    )

    yield

    # Shutdown phase - flush remaining loggers, close httpx client, and thread pool
    if hasattr(app.state, 'loggers'):
        for handler in list(app.state.loggers):
            try:
                await handler.async_flush()
                await handler.close()
            except Exception as e:
                print(f"Error cleaning up ActivityLogHandler: {e}")
        app.state.loggers.clear()

    # Close shared httpx client
    if hasattr(app.state, 'httpx_client'):
        await app.state.httpx_client.aclose()

    if hasattr(app.state, 'executor'):
        app.state.executor.shutdown(wait=True)


def create_app(handler):
    app = FastAPI(title=os.environ.get('FUNCTION_NAME', 'default-function-name'), lifespan=lifespan)

    @app.post("/")
    @log
    async def handle(request: Request, logger=None):
        try:
            payload = await request.json()

            payload["metadata"] = {
                "activityID": request.headers.get(ActivityIDHeader),
                "environmentID": request.headers.get(EnvironmentIDHeader),
                "environmentName": request.headers.get(EnvironmentNameHeader),
                "organizationID": request.headers.get(OrganizationIDHeader),
                "projectID": request.headers.get(ProjectIDHeader),
                "stateStoreUrl": request.headers.get(EaasStateEndpointHeader),
                "stateStoreToken": request.headers.get(EaasStateAPITokenHeader),
            }

            # If the handler is async → await it directly
            if inspect.iscoroutinefunction(handler):
                resp = await handler(logger, payload)
            else:
                # If the handler is sync → run it in a thread pool
                loop = asyncio.get_running_loop()
                resp = await loop.run_in_executor(
                    app.state.executor,
                    handler,
                    logger,
                    payload,
                )
            return {"data": resp}

        except ExecuteAgainException as e:
            return JSONResponse(e.__dict__, 500)
        except FailedException as e:
            return JSONResponse(e.__dict__, 500)
        except TransientException as e:
            return JSONResponse(e.__dict__, 500)
        except Exception as e:
            return JSONResponse(
                content={"error_code": ERROR_CODE_FAILED, "message": str(e)},
                status_code=500,
            )

    @app.get("/_/ready")
    async def ready():
        return {"status": "ready"}

    return app
