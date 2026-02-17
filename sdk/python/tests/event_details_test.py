"""Tests for EventDetails (mirroring Go SDK types_test.go)."""

import unittest

from python_sdk_rafay_workflow import (
    EventDetails,
    EventType,
)


def request_with_event_metadata(event_source: str, event_source_name: str, event_type: str) -> dict:
    """Build a request dict with metadata populated for event fields."""
    return {
        "metadata": {
            "eventSource": event_source,
            "eventSourceName": event_source_name,
            "eventType": event_type,
        }
    }


class TestNewEventDetails(unittest.TestCase):
    def test_workload_deploy(self):
        req = request_with_event_metadata("workload", "my-app", "deploy")
        event = EventDetails(req)
        self.assertEqual(event.source, "workload")
        self.assertEqual(event.source_name, "my-app")
        self.assertEqual(event.type, EventType.DEPLOY)

    def test_environment_destroy(self):
        req = request_with_event_metadata("environment", "prod", "destroy")
        event = EventDetails(req)
        self.assertEqual(event.source, "environment")
        self.assertEqual(event.source_name, "prod")
        self.assertEqual(event.type, EventType.DESTROY)

    def test_empty_metadata_keys(self):
        req = request_with_event_metadata("", "", "")
        event = EventDetails(req)
        self.assertEqual(event.source, "")
        self.assertEqual(event.source_name, "")
        self.assertIsNone(event.type)

    def test_missing_metadata(self):
        event = EventDetails({})
        self.assertEqual(event.source, "")
        self.assertEqual(event.source_name, "")
        self.assertIsNone(event.type)


class TestEventDetailsGetActionName(unittest.TestCase):
    def test_action_source(self):
        event = EventDetails(request_with_event_metadata("action", "run-me", "deploy"))
        name, ok = event.get_action_name()
        self.assertTrue(ok)
        self.assertEqual(name, "run-me")

    def test_wrong_source(self):
        event = EventDetails(request_with_event_metadata("workload", "app", "deploy"))
        name, ok = event.get_action_name()
        self.assertFalse(ok)
        self.assertEqual(name, "")


class TestEventDetailsIsAction(unittest.TestCase):
    def test_action(self):
        self.assertTrue(EventDetails(request_with_event_metadata("action", "", "")).is_action())

    def test_workload(self):
        self.assertFalse(EventDetails(request_with_event_metadata("workload", "", "")).is_action())


class TestEventDetailsGetSchedulesName(unittest.TestCase):
    def test_schedules_source(self):
        event = EventDetails(request_with_event_metadata("schedules", "cron-1", "deploy"))
        name, ok = event.get_schedules_name()
        self.assertTrue(ok)
        self.assertEqual(name, "cron-1")

    def test_wrong_source(self):
        event = EventDetails(request_with_event_metadata("action", "x", "deploy"))
        name, ok = event.get_schedules_name()
        self.assertFalse(ok)
        self.assertEqual(name, "")


class TestEventDetailsIsSchedules(unittest.TestCase):
    def test_schedules(self):
        self.assertTrue(EventDetails(request_with_event_metadata("schedules", "", "")).is_schedules())

    def test_other(self):
        self.assertFalse(EventDetails(request_with_event_metadata("workload", "", "")).is_schedules())


class TestEventDetailsGetWorkloadName(unittest.TestCase):
    def test_workload_source(self):
        event = EventDetails(request_with_event_metadata("workload", "api", "deploy"))
        name, ok = event.get_workload_name()
        self.assertTrue(ok)
        self.assertEqual(name, "api")

    def test_wrong_source(self):
        event = EventDetails(request_with_event_metadata("environment", "prod", "destroy"))
        name, ok = event.get_workload_name()
        self.assertFalse(ok)
        self.assertEqual(name, "")


class TestEventDetailsIsWorkload(unittest.TestCase):
    def test_workload(self):
        self.assertTrue(EventDetails(request_with_event_metadata("workload", "", "")).is_workload())

    def test_other(self):
        self.assertFalse(EventDetails(request_with_event_metadata("environment", "", "")).is_workload())


class TestEventDetailsIsDeploy(unittest.TestCase):
    def test_deploy(self):
        self.assertTrue(EventDetails(request_with_event_metadata("", "", EventType.DEPLOY.value)).is_deploy())

    def test_destroy(self):
        self.assertFalse(EventDetails(request_with_event_metadata("", "", EventType.DESTROY.value)).is_deploy())

    def test_force_destroy(self):
        self.assertFalse(EventDetails(request_with_event_metadata("", "", EventType.FORCE_DESTROY.value)).is_deploy())


class TestEventDetailsIsDestroy(unittest.TestCase):
    def test_destroy(self):
        self.assertTrue(EventDetails(request_with_event_metadata("", "", EventType.DESTROY.value)).is_destroy())

    def test_force_destroy(self):
        self.assertTrue(EventDetails(request_with_event_metadata("", "", EventType.FORCE_DESTROY.value)).is_destroy())

    def test_deploy(self):
        self.assertFalse(EventDetails(request_with_event_metadata("", "", EventType.DEPLOY.value)).is_destroy())


class TestEventDetailsIsForceDestroy(unittest.TestCase):
    def test_force_destroy(self):
        self.assertTrue(EventDetails(request_with_event_metadata("", "", EventType.FORCE_DESTROY.value)).is_force_destroy())

    def test_destroy(self):
        self.assertFalse(EventDetails(request_with_event_metadata("", "", EventType.DESTROY.value)).is_force_destroy())


class TestEventDetailsGetTypeAsString(unittest.TestCase):
    def test_deploy(self):
        self.assertEqual(
            EventDetails(request_with_event_metadata("", "", EventType.DEPLOY.value)).get_type_as_string(),
            "deploy",
        )

    def test_force_destroy(self):
        self.assertEqual(
            EventDetails(request_with_event_metadata("", "", EventType.FORCE_DESTROY.value)).get_type_as_string(),
            "force-destroy",
        )


class TestEventDetailsIsWorkloadDeploy(unittest.TestCase):
    def test_workload_deploy(self):
        self.assertTrue(
            EventDetails(request_with_event_metadata("workload", "", EventType.DEPLOY.value)).is_workload_deploy()
        )

    def test_workload_destroy(self):
        self.assertFalse(
            EventDetails(request_with_event_metadata("workload", "", EventType.DESTROY.value)).is_workload_deploy()
        )

    def test_environment_deploy(self):
        self.assertFalse(
            EventDetails(request_with_event_metadata("environment", "", EventType.DEPLOY.value)).is_workload_deploy()
        )


class TestEventDetailsIsWorkloadDestroy(unittest.TestCase):
    def test_workload_destroy(self):
        self.assertTrue(
            EventDetails(request_with_event_metadata("workload", "", EventType.DESTROY.value)).is_workload_destroy()
        )

    def test_workload_force_destroy(self):
        self.assertTrue(
            EventDetails(request_with_event_metadata("workload", "", EventType.FORCE_DESTROY.value)).is_workload_destroy()
        )

    def test_workload_deploy(self):
        self.assertFalse(
            EventDetails(request_with_event_metadata("workload", "", EventType.DEPLOY.value)).is_workload_destroy()
        )

    def test_environment_destroy(self):
        self.assertFalse(
            EventDetails(request_with_event_metadata("environment", "", EventType.DESTROY.value)).is_workload_destroy()
        )


class TestEventDetailsIsEnvironmentDeploy(unittest.TestCase):
    def test_environment_deploy(self):
        self.assertTrue(
            EventDetails(request_with_event_metadata("environment", "", EventType.DEPLOY.value)).is_environment_deploy()
        )

    def test_environment_destroy(self):
        self.assertFalse(
            EventDetails(request_with_event_metadata("environment", "", EventType.DESTROY.value)).is_environment_deploy()
        )

    def test_workload_deploy(self):
        self.assertFalse(
            EventDetails(request_with_event_metadata("workload", "", EventType.DEPLOY.value)).is_environment_deploy()
        )


class TestEventDetailsIsEnvironmentDestroy(unittest.TestCase):
    def test_environment_destroy(self):
        self.assertTrue(
            EventDetails(request_with_event_metadata("environment", "", EventType.DESTROY.value)).is_environment_destroy()
        )

    def test_environment_force_destroy(self):
        self.assertTrue(
            EventDetails(request_with_event_metadata("environment", "", EventType.FORCE_DESTROY.value)).is_environment_destroy()
        )

    def test_environment_deploy(self):
        self.assertFalse(
            EventDetails(request_with_event_metadata("environment", "", EventType.DEPLOY.value)).is_environment_destroy()
        )

    def test_workload_destroy(self):
        self.assertFalse(
            EventDetails(request_with_event_metadata("workload", "", EventType.DESTROY.value)).is_environment_destroy()
        )


class TestEventTypeEnum(unittest.TestCase):
    def test_enum_values_and_str(self):
        self.assertEqual(str(EventType.DEPLOY), "deploy")
        self.assertEqual(str(EventType.DESTROY), "destroy")
        self.assertEqual(str(EventType.FORCE_DESTROY), "force-destroy")
        self.assertEqual(EventType.DEPLOY.value, "deploy")
        self.assertEqual(EventType.DESTROY.value, "destroy")
        self.assertEqual(EventType.FORCE_DESTROY.value, "force-destroy")


if __name__ == "__main__":
    unittest.main()
