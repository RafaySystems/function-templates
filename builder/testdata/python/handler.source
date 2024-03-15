from typing import *
from logging import Logger

def handle(logger: Logger,request: Dict[str, Any]) -> Dict[str, Any]:
    logger.info(f"inside function handler, request: {request}", extra={"request": request})
    return {
       "message": "Hello, World!"
    }
