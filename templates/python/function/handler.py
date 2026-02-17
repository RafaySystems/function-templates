from logging import Logger
from typing import Dict, Any

from python_sdk_rafay_workflow import sdk, EventDetails


async def handle(logger: Logger, request: Dict[str, Any]) -> Dict[str, Any]:
    logger.info(f"received request, req: {request}")

    if request.get("fail", False):
        raise sdk.FailedException("failed!!!")

    event = EventDetails(request=request)

    return {
        "hell": "python",
        "source": event.source,
        "sourceName": event.source_name,
        "type": event.type,
    }
