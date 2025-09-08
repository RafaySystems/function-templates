import asyncio
from logging import Logger
from typing import Dict, Any

from python_sdk_rafay_workflow import sdk


async def handle(logger: Logger, request: Dict[str, Any]) -> Dict[str, Any]:
    logger.info(f"received request, req: {request}")

    if request.get("fail", False):
        raise sdk.FailedException("failed!!!")

    for i in range(request.get("sleep", 1)):
        logger.info(f"iteration {i + 1}: sleeping for 1 minute")
        await asyncio.sleep(60) # Use time.sleep for synchronous sleep

    return {
        "output": "Hello Python!",
    }
