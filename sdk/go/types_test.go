package sdk_test

import (
	"testing"

	sdk "github.com/RafaySystems/function-templates/sdk/go"
	"github.com/google/go-cmp/cmp"
)

// requestWithEventMetadata returns a Request with metadata populated for event fields.
func requestWithEventMetadata(eventSource, eventSourceName, eventType string) sdk.Request {
	req := make(sdk.Request)
	req["metadata"] = map[string]string{
		"eventSource":     eventSource,
		"eventSourceName": eventSourceName,
		"eventType":       eventType,
	}
	return req
}

func TestNewEventDetails(t *testing.T) {
	tests := []struct {
		name           string
		req            sdk.Request
		wantSource     string
		wantSourceName string
		wantType       sdk.EventType
	}{
		{
			name:           "workload deploy",
			req:            requestWithEventMetadata("workload", "my-app", "deploy"),
			wantSource:     "workload",
			wantSourceName: "my-app",
			wantType:       sdk.DeployEventType,
		},
		{
			name:           "environment destroy",
			req:            requestWithEventMetadata("environment", "prod", "destroy"),
			wantSource:     "environment",
			wantSourceName: "prod",
			wantType:       sdk.DestroyEventType,
		},
		{
			name:           "action source",
			req:            requestWithEventMetadata("action", "my-action", "deploy"),
			wantSource:     "action",
			wantSourceName: "my-action",
			wantType:       sdk.DeployEventType,
		},
		{
			name:           "schedules source",
			req:            requestWithEventMetadata("schedules", "daily-job", "deploy"),
			wantSource:     "schedules",
			wantSourceName: "daily-job",
			wantType:       sdk.DeployEventType,
		},
		{
			name:           "force-destroy type",
			req:            requestWithEventMetadata("environment", "stale", "force-destroy"),
			wantSource:     "environment",
			wantSourceName: "stale",
			wantType:       sdk.ForceDestroyEventType,
		},
		{
			name:           "empty metadata keys",
			req:            requestWithEventMetadata("", "", ""),
			wantSource:     "",
			wantSourceName: "",
			wantType:       "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sdk.NewEventDetails(tt.req)
			if got == nil {
				t.Fatal("NewEventDetails returned nil")
			}
			if got.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", got.Source, tt.wantSource)
			}
			if got.SourceName != tt.wantSourceName {
				t.Errorf("SourceName = %q, want %q", got.SourceName, tt.wantSourceName)
			}
			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
		})
	}
}

func TestEventDetails_GetActionName(t *testing.T) {
	tests := []struct {
		name        string
		event       sdk.EventDetails
		wantName    string
		wantOk      bool
	}{
		{"action source", sdk.EventDetails{Source: "action", SourceName: "run-me"}, "run-me", true},
		{"wrong source", sdk.EventDetails{Source: "workload", SourceName: "app"}, "", false},
		{"empty source name", sdk.EventDetails{Source: "action", SourceName: ""}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotOk := tt.event.GetActionName()
			if gotOk != tt.wantOk || gotName != tt.wantName {
				t.Errorf("GetActionName() = (%q, %v), want (%q, %v)", gotName, gotOk, tt.wantName, tt.wantOk)
			}
		})
	}
}

func TestEventDetails_IsAction(t *testing.T) {
	tests := []struct {
		name  string
		event sdk.EventDetails
		want  bool
	}{
		{"action", sdk.EventDetails{Source: "action"}, true},
		{"workload", sdk.EventDetails{Source: "workload"}, false},
		{"environment", sdk.EventDetails{Source: "environment"}, false},
		{"empty", sdk.EventDetails{Source: ""}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.IsAction(); got != tt.want {
				t.Errorf("IsAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventDetails_GetSchedulesName(t *testing.T) {
	tests := []struct {
		name     string
		event    sdk.EventDetails
		wantName string
		wantOk   bool
	}{
		{"schedules source", sdk.EventDetails{Source: "schedules", SourceName: "cron-1"}, "cron-1", true},
		{"wrong source", sdk.EventDetails{Source: "action", SourceName: "x"}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotOk := tt.event.GetSchedulesName()
			if gotOk != tt.wantOk || gotName != tt.wantName {
				t.Errorf("GetSchedulesName() = (%q, %v), want (%q, %v)", gotName, gotOk, tt.wantName, tt.wantOk)
			}
		})
	}
}

func TestEventDetails_IsSchedules(t *testing.T) {
	tests := []struct {
		name  string
		event sdk.EventDetails
		want  bool
	}{
		{"schedules", sdk.EventDetails{Source: "schedules"}, true},
		{"other", sdk.EventDetails{Source: "workload"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.IsSchedules(); got != tt.want {
				t.Errorf("IsSchedules() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventDetails_GetWorkloadName(t *testing.T) {
	tests := []struct {
		name     string
		event    sdk.EventDetails
		wantName string
		wantOk   bool
	}{
		{"workload source", sdk.EventDetails{Source: "workload", SourceName: "api"}, "api", true},
		{"wrong source", sdk.EventDetails{Source: "environment", SourceName: "prod"}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotOk := tt.event.GetWorkloadName()
			if gotOk != tt.wantOk || gotName != tt.wantName {
				t.Errorf("GetWorkloadName() = (%q, %v), want (%q, %v)", gotName, gotOk, tt.wantName, tt.wantOk)
			}
		})
	}
}

func TestEventDetails_IsWorkload(t *testing.T) {
	tests := []struct {
		name  string
		event sdk.EventDetails
		want  bool
	}{
		{"workload", sdk.EventDetails{Source: "workload"}, true},
		{"other", sdk.EventDetails{Source: "environment"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.IsWorkload(); got != tt.want {
				t.Errorf("IsWorkload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventDetails_IsDeploy(t *testing.T) {
	tests := []struct {
		name  string
		event sdk.EventDetails
		want  bool
	}{
		{"deploy", sdk.EventDetails{Type: sdk.DeployEventType}, true},
		{"destroy", sdk.EventDetails{Type: sdk.DestroyEventType}, false},
		{"force-destroy", sdk.EventDetails{Type: sdk.ForceDestroyEventType}, false},
		{"empty", sdk.EventDetails{Type: ""}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.IsDeploy(); got != tt.want {
				t.Errorf("IsDeploy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventDetails_IsDestroy(t *testing.T) {
	tests := []struct {
		name  string
		event sdk.EventDetails
		want  bool
	}{
		{"destroy", sdk.EventDetails{Type: sdk.DestroyEventType}, true},
		{"force-destroy", sdk.EventDetails{Type: sdk.ForceDestroyEventType}, true},
		{"deploy", sdk.EventDetails{Type: sdk.DeployEventType}, false},
		{"empty", sdk.EventDetails{Type: ""}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.IsDestroy(); got != tt.want {
				t.Errorf("IsDestroy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventDetails_IsForceDestroy(t *testing.T) {
	tests := []struct {
		name  string
		event sdk.EventDetails
		want  bool
	}{
		{"force-destroy", sdk.EventDetails{Type: sdk.ForceDestroyEventType}, true},
		{"destroy", sdk.EventDetails{Type: sdk.DestroyEventType}, false},
		{"deploy", sdk.EventDetails{Type: sdk.DeployEventType}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.IsForceDestroy(); got != tt.want {
				t.Errorf("IsForceDestroy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventDetails_GetTypeAsString(t *testing.T) {
	tests := []struct {
		name  string
		event sdk.EventDetails
		want  string
	}{
		{"deploy", sdk.EventDetails{Type: sdk.DeployEventType}, "deploy"},
		{"destroy", sdk.EventDetails{Type: sdk.DestroyEventType}, "destroy"},
		{"force-destroy", sdk.EventDetails{Type: sdk.ForceDestroyEventType}, "force-destroy"},
		{"empty", sdk.EventDetails{Type: ""}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.GetTypeAsString(); got != tt.want {
				t.Errorf("GetTypeAsString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEventDetails_IsWorkloadDeploy(t *testing.T) {
	tests := []struct {
		name  string
		event sdk.EventDetails
		want  bool
	}{
		{"workload + deploy", sdk.EventDetails{Source: "workload", Type: sdk.DeployEventType}, true},
		{"workload + destroy", sdk.EventDetails{Source: "workload", Type: sdk.DestroyEventType}, false},
		{"environment + deploy", sdk.EventDetails{Source: "environment", Type: sdk.DeployEventType}, false},
		{"wrong source", sdk.EventDetails{Source: "action", Type: sdk.DeployEventType}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.IsWorkloadDeploy(); got != tt.want {
				t.Errorf("IsWorkloadDeploy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventDetails_IsWorkloadDestroy(t *testing.T) {
	tests := []struct {
		name  string
		event sdk.EventDetails
		want  bool
	}{
		{"workload + destroy", sdk.EventDetails{Source: "workload", Type: sdk.DestroyEventType}, true},
		{"workload + force-destroy", sdk.EventDetails{Source: "workload", Type: sdk.ForceDestroyEventType}, true},
		{"workload + deploy", sdk.EventDetails{Source: "workload", Type: sdk.DeployEventType}, false},
		{"environment + destroy", sdk.EventDetails{Source: "environment", Type: sdk.DestroyEventType}, false},
		{"environment + force-destroy", sdk.EventDetails{Source: "environment", Type: sdk.ForceDestroyEventType}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.IsWorkloadDestroy(); got != tt.want {
				t.Errorf("IsWorkloadDestroy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventDetails_IsEnvironmentDeploy(t *testing.T) {
	tests := []struct {
		name  string
		event sdk.EventDetails
		want  bool
	}{
		{"environment + deploy", sdk.EventDetails{Source: "environment", Type: sdk.DeployEventType}, true},
		{"environment + destroy", sdk.EventDetails{Source: "environment", Type: sdk.DestroyEventType}, false},
		{"workload + deploy", sdk.EventDetails{Source: "workload", Type: sdk.DeployEventType}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.IsEnvironmentDeploy(); got != tt.want {
				t.Errorf("IsEnvironmentDeploy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventDetails_IsEnvironmentDestroy(t *testing.T) {
	tests := []struct {
		name  string
		event sdk.EventDetails
		want  bool
	}{
		{"environment + destroy", sdk.EventDetails{Source: "environment", Type: sdk.DestroyEventType}, true},
		{"environment + force-destroy", sdk.EventDetails{Source: "environment", Type: sdk.ForceDestroyEventType}, true},
		{"environment + deploy", sdk.EventDetails{Source: "environment", Type: sdk.DeployEventType}, false},
		{"workload + destroy", sdk.EventDetails{Source: "workload", Type: sdk.DestroyEventType}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.IsEnvironmentDestroy(); got != tt.want {
				t.Errorf("IsEnvironmentDestroy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventType_constants(t *testing.T) {
	// Ensure event type constants have expected string values.
	if diff := cmp.Diff(string(sdk.DeployEventType), "deploy"); diff != "" {
		t.Errorf("DeployEventType: %s", diff)
	}
	if diff := cmp.Diff(string(sdk.DestroyEventType), "destroy"); diff != "" {
		t.Errorf("DestroyEventType: %s", diff)
	}
	if diff := cmp.Diff(string(sdk.ForceDestroyEventType), "force-destroy"); diff != "" {
		t.Errorf("ForceDestroyEventType: %s", diff)
	}
}
