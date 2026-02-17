// Package sdk Adapted from https://github.com/openfaas/templates-sdk/blob/master/go-http/handler.go
// Original license: MIT
package sdk

import (
	"context"
	"log/slog"
)

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Log(ctx context.Context, level slog.Level, msg string, args ...any)
}

type Object map[string]any

func (r Object) MetaString(key string) string {
	return r["metadata"].(map[string]string)[key]
}

type (
	Request  = Object
	Response = Object
)

func (r Object) GetAsString(key string) (string, bool) {
	val, ok := r[key].(string)
	return val, ok
}

// FunctionHandler used for a serverless Go method invocation
type FunctionHandler interface {
	Handle(ctx context.Context, logger Logger, req Request) (Response, error)
}

type Handler func(ctx context.Context, logger Logger, req Request) (Response, error)

const (
	ActivityIDHeader         = "X-Activity-ID"
	EnvironmentIDHeader      = "X-Environment-ID"
	EnvironmentNameHeader    = "X-Environment-Name"
	WorkflowTokenHeader      = "X-Workflow-Token"
	EngineAPIEndpointHeader  = "X-Engine-Endpoint"
	ActivityFileUploadHeader = "X-Activity-File-Upload"
	OrganizationIDHeader     = "X-Organization-ID"
	ProjectIDHeader          = "X-Project-ID"
	EaasStateEndpointHeader  = "X-Eaas-State-Endpoint"
	EaasStateAPITokenHeader  = "X-Eaas-State-Token"
	EventSourceHeader        = "X-Event-Source"
	EventSourceNameHeader    = "X-Event-Source-Name"
	EventTypeHeader          = "X-Event-Type"
)

type ReadyResponse struct {
	Ready          bool  `json:"ready"`
	NumConnections int32 `json:"num_connections"`
}

// EventType represents the type of event, commonly used in event-driven systems for categorization or processing.
type EventType string

const (
	DeployEventType       EventType = "deploy"
	DestroyEventType      EventType = "destroy"
	ForceDestroyEventType EventType = "force-destroy"
)

// EventDetails represents metadata about an event, including its source, source name, and type.
type EventDetails struct {
	Source     string
	SourceName string
	Type       EventType
}

// NewEventDetails creates a new EventDetails instance by extracting metadata fields from the given Request object.
// The request's metadata is used to populate the EventDetails's Source, SourceName, and Type fields.
func NewEventDetails(request Request) *EventDetails {
	return &EventDetails{
		Source:     request.MetaString("eventSource"),
		SourceName: request.MetaString("eventSourceName"),
		Type:       EventType(request.MetaString("eventType")),
	}
}

// getSourceName retrieves the SourceName if the event's Source matches the provided source
func (e EventDetails) getSourceName(source string) (string, bool) {
	if e.Source != source {
		return "", false
	}
	return e.SourceName, true
}

// isSource checks whether the event's Source matches the provided source string.
func (e EventDetails) isSource(source string) bool {
	return e.Source == source
}

// GetActionName returns the SourceName and true if the event Source is "action";
// otherwise returns an empty string and false.
func (e EventDetails) GetActionName() (string, bool) {
	return e.getSourceName("action")
}

// IsAction determines if the event's Source is set to "action".
func (e EventDetails) IsAction() bool {
	return e.isSource("action")
}

// GetSchedulesName returns the SourceName and true if the event Source is "schedules";
// otherwise, returns an empty string and false.
func (e EventDetails) GetSchedulesName() (string, bool) {
	return e.getSourceName("schedules")
}

// IsSchedules determines if the event's Source is set to "schedules".
func (e EventDetails) IsSchedules() bool {
	return e.isSource("schedules")
}

// GetWorkloadName returns the SourceName and true if the event Source is "workload";
// otherwise, returns an empty string and false.
func (e EventDetails) GetWorkloadName() (string, bool) {
	return e.getSourceName("workload")
}

// IsWorkload checks if the event's Source is set to "workload".
func (e EventDetails) IsWorkload() bool {
	return e.isSource("workload")
}

// IsWorkloadDeploy checks if the event's Source is "workload" and its Type is DeployEventType.
func (e EventDetails) IsWorkloadDeploy() bool {
	return e.isSource("workload") && e.IsDeploy()
}

// IsWorkloadDestroy returns true if the event's Source is "environment" and its Type indicates a destroy operation.
func (e EventDetails) IsWorkloadDestroy() bool {
	return e.isSource("workload") && e.IsDestroy()
}

// IsEnvironmentDeploy checks if the event's Source is "environment" and its Type is DeployEventType.
func (e EventDetails) IsEnvironmentDeploy() bool {
	return e.isSource("environment") && e.IsDeploy()
}

// IsEnvironmentDestroy checks if the event's Source is "environment" and its Type is DestroyEventType or ForceDestroyEventType.
func (e EventDetails) IsEnvironmentDestroy() bool {
	return e.isSource("environment") && e.IsDestroy()
}

// IsDeploy checks if the event's Type is DeployEventType.
func (e EventDetails) IsDeploy() bool {
	return e.Type == DeployEventType
}

// IsDestroy checks if the event's Type is set to DestroyEventType or ForceDestroyEventType.
func (e EventDetails) IsDestroy() bool {
	return e.Type == DestroyEventType || e.Type == ForceDestroyEventType
}

// IsForceDestroy checks if the event's Type is set to ForceDestroyEventType.
func (e EventDetails) IsForceDestroy() bool {
	return e.Type == ForceDestroyEventType
}

// GetTypeAsString converts the EventDetails's Type field to a string and returns it.
func (e EventDetails) GetTypeAsString() string {
	return string(e.Type)
}
