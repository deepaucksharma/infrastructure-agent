package watchdog

import (
	"sync"
	"time"
)

// DiagnosticEvent represents a diagnostic event
type DiagnosticEvent struct {
	// ID is a unique identifier for the event
	ID string
	
	// Type is the type of event
	Type string
	
	// ComponentName is the name of the component that generated the event
	ComponentName string
	
	// Timestamp is when the event occurred
	Timestamp time.Time
	
	// Severity is the severity level of the event
	Severity string
	
	// Message is a human-readable message describing the event
	Message string
	
	// Details contains additional details about the event
	Details map[string]interface{}
}

// DiagnosticsProvider records and provides diagnostic events
type DiagnosticsProvider struct {
	// events contains the recorded events
	events []DiagnosticEvent
	
	// maxEvents is the maximum number of events to retain
	maxEvents int
	
	// includeStackTraces indicates whether to include stack traces in events
	includeStackTraces bool
	
	// mutex protects the events slice
	mutex sync.RWMutex
}

// NewDiagnosticsProvider creates a new diagnostics provider
func NewDiagnosticsProvider() *DiagnosticsProvider {
	return &DiagnosticsProvider{
		events:            make([]DiagnosticEvent, 0, 100),
		maxEvents:         100,
		includeStackTraces: true,
	}
}

// EmitAgentDiagEvent records an agent diagnostic event
func (d *DiagnosticsProvider) EmitAgentDiagEvent(incident Incident) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	// Convert incident to diagnostic event
	event := DiagnosticEvent{
		ID:            incident.ID,
		Type:          string(incident.Type),
		ComponentName: "",  // Will be filled by caller
		Timestamp:     incident.Timestamp,
		Severity:      incidentSeverity(incident.Type),
		Message:       incident.Description,
		Details:       make(map[string]interface{}),
	}
	
	// Add resource usage details
	event.Details["cpu_percent"] = incident.ResourceUsage.CPUPercent
	event.Details["memory_mb"] = incident.ResourceUsage.MemoryMB
	event.Details["goroutines"] = incident.ResourceUsage.Goroutines
	event.Details["file_handles"] = incident.ResourceUsage.FileHandles
	event.Details["gc_percent"] = incident.ResourceUsage.GCPercent
	
	// Add remediation
	if incident.Remediation != "" {
		event.Details["remediation"] = incident.Remediation
	}
	
	// Add to events list
	d.events = append(d.events, event)
	
	// Trim if needed
	if len(d.events) > d.maxEvents {
		d.events = d.events[len(d.events)-d.maxEvents:]
	}
}

// GetEvents returns all recorded events
func (d *DiagnosticsProvider) GetEvents() []DiagnosticEvent {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	// Create a copy to prevent modification
	events := make([]DiagnosticEvent, len(d.events))
	copy(events, d.events)
	
	return events
}

// GetEventsByComponent returns events for a specific component
func (d *DiagnosticsProvider) GetEventsByComponent(componentName string) []DiagnosticEvent {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	var events []DiagnosticEvent
	
	for _, event := range d.events {
		if event.ComponentName == componentName {
			events = append(events, event)
		}
	}
	
	return events
}

// GetEventsByType returns events of a specific type
func (d *DiagnosticsProvider) GetEventsByType(eventType string) []DiagnosticEvent {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	
	var events []DiagnosticEvent
	
	for _, event := range d.events {
		if event.Type == eventType {
			events = append(events, event)
		}
	}
	
	return events
}

// ClearEvents clears all recorded events
func (d *DiagnosticsProvider) ClearEvents() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	d.events = make([]DiagnosticEvent, 0, d.maxEvents)
}

// SetMaxEvents sets the maximum number of events to retain
func (d *DiagnosticsProvider) SetMaxEvents(maxEvents int) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	d.maxEvents = maxEvents
	
	// Trim if needed
	if len(d.events) > d.maxEvents {
		d.events = d.events[len(d.events)-d.maxEvents:]
	}
}

// SetIncludeStackTraces sets whether to include stack traces in events
func (d *DiagnosticsProvider) SetIncludeStackTraces(include bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	d.includeStackTraces = include
}

// incidentSeverity returns the severity level for an incident type
func incidentSeverity(incidentType IncidentType) string {
	switch incidentType {
	case IncidentResourceExceeded:
		return "warning"
	case IncidentDeadlockDetected:
		return "critical"
	case IncidentRestartFailed:
		return "critical"
	case IncidentCrash:
		return "critical"
	default:
		return "info"
	}
}
