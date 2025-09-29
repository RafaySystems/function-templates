import asyncio
from logging import Logger
from typing import Dict, Any

from python_sdk_rafay_workflow import sdk
from python_sdk_rafay_workflow.state_client import StateClientBuilder


async def handle(logger: Logger, request: Dict[str, Any]) -> Dict[str, Any]:
    logger.info(f"received request, req: {request}")

    if request.get("fail", False):
        raise sdk.FailedException("failed!!!")

    # Build state client for organization scope
    ostate = StateClientBuilder(request).with_org_scope()

    # --- SYNC TESTS ---
    # Set (OCC safe)
    ostate.set("org_counter", lambda old: int(old or 0) + 1)
    # SetKV (no OCC)
    value, version = ostate.get("org_counter")
    logger.info(f"org_counter sync get: {value}, version: {version}")
    ostate.set_kv("org_counter", str(int(value or 0) + 1), version)
    # Delete
    ostate.delete("org_counter")
    logger.info("org_counter sync deleted")

    # --- ASYNC TESTS ---
    # SetAsync (OCC safe)
    await ostate.set_async("org_counter_async", lambda old: int(old or 0) + 1)
    # SetKVAsync (no OCC)
    value_async, version_async = await ostate.get_async("org_counter_async")
    logger.info(f"org_counter_async async get: {value_async}, version: {version_async}")
    await ostate.set_kv_async("org_counter_async", str(int(value_async or 0) + 1), version_async)
    # DeleteAsync
    await ostate.delete_async("org_counter_async")
    logger.info("org_counter_async async deleted")

    # --- PROJECT SCOPE ---
    pstate = StateClientBuilder(request).with_project_scope()
    pstate.set("project_counter", lambda old: int(old or 0) + 1)
    value, version = pstate.get("project_counter")
    logger.info(f"project_counter sync get: {value}, version: {version}")
    pstate.set_kv("project_counter", str(int(value or 0) + 1), version)
    pstate.delete("project_counter")
    logger.info("project_counter sync deleted")
    
    # --- ASYNC TESTS ---
    await pstate.set_async("project_counter_async", lambda old: int(old or 0) + 1)
    value_async, version_async = await pstate.get_async("project_counter_async")
    logger.info(f"project_counter_async async get: {value_async}, version: {version_async}")
    await pstate.set_kv_async("project_counter_async", str(int(value_async or 0) + 1), version_async)
    await pstate.delete_async("project_counter_async")
    logger.info("project_counter_async async deleted")

    # --- ENV SCOPE ---
    state = StateClientBuilder(request).with_env_scope()
    state.set("env_counter", lambda old: int(old or 0) + 1)
    value, version = state.get("env_counter")
    logger.info(f"env_counter sync get: {value}, version: {version}")
    state.set_kv("env_counter", str(int(value or 0) + 1), version)
    state.delete("env_counter")
    logger.info("env_counter sync deleted")
    
    # --- ASYNC TESTS ---
    await state.set_async("env_counter_async", lambda old: int(old or 0) + 1)
    value_async, version_async = await state.get_async("env_counter_async")
    logger.info(f"env_counter_async async get: {value_async}, version: {version_async}")
    await state.set_kv_async("env_counter_async", str(int(value_async or 0) + 1), version_async)
    await state.delete_async("env_counter_async")
    logger.info("env_counter_async async deleted")

    for i in range(request.get("sleep", 1)):
        logger.info(f"iteration {i + 1}: sleeping for 1 minute")
        await asyncio.sleep(60) # Use time.sleep for synchronous sleep

    return {
        "output": "Hello Python!",
    }
