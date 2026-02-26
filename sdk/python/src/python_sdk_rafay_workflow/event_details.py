"""Event metadata for workflow invocations: source, source name, and event type."""

from enum import Enum
from typing import Any, Optional, Tuple


class EventType(str, Enum):
    """Event type (deploy, destroy, force-destroy). Matches Go SDK."""

    DEPLOY = "deploy"
    DESTROY = "destroy"
    FORCE_DESTROY = "force-destroy"

    def __str__(self) -> str:
        """Return the event type as a string."""
        return self.value


class EventDetails:
    """
    Metadata about an event: source, source name, and type.
    Construct from the handler request dict: EventDetails(request).
    """

    source: str
    source_name: str
    type: Optional[EventType]  # event type: deploy, destroy, force-destroy; None if missing or unknown

    def __init__(self, request: dict[str, Any]) -> None:
        metadata = request.get("metadata", {})
        self.source = metadata.get("eventSource") or ""
        self.source_name = metadata.get("eventSourceName") or ""
        try:
            self.type = EventType(metadata.get("eventType"))
        except ValueError:
            self.type = None

    def _get_source_name(self, source: str) -> Tuple[str, bool]:
        if self.source != source:
            return "", False
        return self.source_name, True

    def _is_source(self, source: str) -> bool:
        return self.source == source

    def get_action_name(self) -> Tuple[str, bool]:
        """Return (source_name, True) if source is 'action', else ('', False)."""
        return self._get_source_name("action")

    def is_action(self) -> bool:
        """Return True if event source is 'action'."""
        return self._is_source("action")

    def get_schedules_name(self) -> Tuple[str, bool]:
        """Return (source_name, True) if source is 'schedules', else ('', False)."""
        return self._get_source_name("schedules")

    def is_schedules(self) -> bool:
        """Return True if event source is 'schedules'."""
        return self._is_source("schedules")

    def get_workload_name(self) -> Tuple[str, bool]:
        """Return (source_name, True) if source is 'workload', else ('', False)."""
        return self._get_source_name("workload")

    def is_workload(self) -> bool:
        """Return True if event source is 'workload'."""
        return self._is_source("workload")

    def is_deploy(self) -> bool:
        """Return True if event type is deploy."""
        return self.type == EventType.DEPLOY

    def is_destroy(self) -> bool:
        """Return True if event type is destroy or force-destroy."""
        return self.type in (EventType.DESTROY, EventType.FORCE_DESTROY)

    def is_force_destroy(self) -> bool:
        """Return True if event type is force-destroy."""
        return self.type == EventType.FORCE_DESTROY

    def get_type_as_string(self) -> str:
        """Return the event type as a string."""
        return str(self.type) if self.type is not None else ""

    def is_workload_deploy(self) -> bool:
        """Return True if source is 'workload' and type is deploy."""
        return self._is_source("workload") and self.is_deploy()

    def is_workload_destroy(self) -> bool:
        """Return True if source is 'workload' and type is destroy or force-destroy."""
        return self._is_source("workload") and self.is_destroy()

    def is_environment_deploy(self) -> bool:
        """Return True if source is 'environment' and type is deploy."""
        return self._is_source("environment") and self.is_deploy()

    def is_environment_destroy(self) -> bool:
        """Return True if source is 'environment' and type is destroy or force-destroy."""
        return self._is_source("environment") and self.is_destroy()
