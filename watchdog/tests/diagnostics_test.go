package tests

import (
	"testing"
	"time"

	"github.com/newrelic/infrastructure-agent/watchdog"
	"github.com/stretchr/testify/assert"
)

// TestDiagnosticsProviderCreation tests creating a diagnostics provider
func TestDiagnosticsProviderCreation(t *testing.T) {
	provider := watchdog.NewDiagnosticsProvider()
	assert.NotNil(t, provider)
	
	// Should start with no events
	events := provider.GetEvents()
	assert.Empty(t, events)
}

// TestEmitAgentDiagEvent tests recording agent diagnostic events
func TestEmitAgentDiagEvent(t *testing.T) {
	provider := watchdog.NewDiagnosticsProvider()
	
	// Create an incident
	resourceUsage := watchdog.ResourceUsage{
		CPUPercent:  90.0,
		MemoryMB:    500.0,
		Goroutines:  100,
		FileHandles: 50,
		GCPercent:   5.0,
		Timestamp:   time.Now(),
	}
	
	incident := watchdog.Incident{
		ID:            "test-incident-1",
		Timestamp:     time.Now(),
		Type:          watchdog.IncidentResourceExceeded,
		Description:   "CPU usage exceeded threshold",
		ResourceUsage: resourceUsage,
		Remediation:   "Consider reducing CPU-intensive operations",
	}
	
	// Emit the event
	provider.EmitAgentDiagEvent(incident)
	
	// Check that the event was recorded
	events := provider.GetEvents()
	assert.Len(t, events, 1)
	
	event := events[0]
	assert.Equal(t, incident.ID, event.ID)
	assert.Equal(t, string(incident.Type), event.Type)
	assert.Equal(t, incident.Timestamp, event.Timestamp)
	assert.Equal(t, "warning", event.Severity) // ResourceExceeded is warning level
	assert.Equal(t, incident.Description, event.Message)
	
	// Check details
	assert.Equal(t, incident.ResourceUsage.CPUPercent, event.Details["cpu_percent"])
	assert.Equal(t, incident.ResourceUsage.MemoryMB, event.Details["memory_mb"])
	assert.Equal(t, incident.ResourceUsage.Goroutines, event.Details["goroutines"])
	assert.Equal(t, incident.ResourceUsage.FileHandles, event.Details["file_handles"])
	assert.Equal(t, incident.ResourceUsage.GCPercent, event.Details["gc_percent"])
	assert.Equal(t, incident.Remediation, event.Details["remediation"])
}

// TestMultipleEvents tests recording multiple events
func TestMultipleEvents(t *testing.T) {
	provider := watchdog.NewDiagnosticsProvider()
	
	// Create resource usage for incidents
	resourceUsage := watchdog.ResourceUsage{
		CPUPercent:  90.0,
		MemoryMB:    500.0,
		Goroutines:  100,
		FileHandles: 50,
		GCPercent:   5.0,
		Timestamp:   time.Now(),
	}
	
	// Create and emit multiple incidents
	incidentTypes := []watchdog.IncidentType{
		watchdog.IncidentResourceExceeded,
		watchdog.IncidentDeadlockDetected,
		watchdog.IncidentRestartFailed,
		watchdog.IncidentCrash,
	}
	
	for i, incidentType := range incidentTypes {
		incident := watchdog.Incident{
			ID:            fmt.Sprintf("test-incident-%d", i+1),
			Timestamp:     time.Now(),
			Type:          incidentType,
			Description:   fmt.Sprintf("Incident of type %s", incidentType),
			ResourceUsage: resourceUsage,
			Remediation:   fmt.Sprintf("Remediation for %s", incidentType),
		}
		
		provider.EmitAgentDiagEvent(incident)
	}
	
	// Check that all events were recorded
	events := provider.GetEvents()
	assert.Len(t, events, 4)
	
	// Check that severities were set correctly
	assert.Equal(t, "warning", events[0].Severity)  // ResourceExceeded
	assert.Equal(t, "critical", events[1].Severity) // DeadlockDetected
	assert.Equal(t, "critical", events[2].Severity) // RestartFailed
	assert.Equal(t, "critical", events[3].Severity) // Crash
}

// TestMaxEvents tests the maximum events limit
func TestMaxEvents(t *testing.T) {
	provider := watchdog.NewDiagnosticsProvider()
	
	// Set a small max events limit
	provider.SetMaxEvents(5)
	
	// Create and emit more than the max number of events
	resourceUsage := watchdog.ResourceUsage{
		CPUPercent:  90.0,
		MemoryMB:    500.0,
		Goroutines:  100,
		FileHandles: 50,
		GCPercent:   5.0,
		Timestamp:   time.Now(),
	}
	
	for i := 0; i < 10; i++ {
		incident := watchdog.Incident{
			ID:            fmt.Sprintf("test-incident-%d", i+1),
			Timestamp:     time.Now(),
			Type:          watchdog.IncidentResourceExceeded,
			Description:   fmt.Sprintf("Incident %d", i+1),
			ResourceUsage: resourceUsage,
		}
		
		provider.EmitAgentDiagEvent(incident)
	}
	
	// Check that only max events were retained (the most recent ones)
	events := provider.GetEvents()
	assert.Len(t, events, 5)
	
	// The events should be the last 5 emitted
	for i := 0; i < 5; i++ {
		assert.Equal(t, fmt.Sprintf("test-incident-%d", i+6), events[i].ID)
	}
}

// TestClearEvents tests clearing events
func TestClearEvents(t *testing.T) {
	provider := watchdog.NewDiagnosticsProvider()
	
	// Create and emit some events
	resourceUsage := watchdog.ResourceUsage{
		CPUPercent:  90.0,
		MemoryMB:    500.0,
		Goroutines:  100,
		FileHandles: 50,
		GCPercent:   5.0,
		Timestamp:   time.Now(),
	}
	
	for i := 0; i < 5; i++ {
		incident := watchdog.Incident{
			ID:            fmt.Sprintf("test-incident-%d", i+1),
			Timestamp:     time.Now(),
			Type:          watchdog.IncidentResourceExceeded,
			Description:   fmt.Sprintf("Incident %d", i+1),
			ResourceUsage: resourceUsage,
		}
		
		provider.EmitAgentDiagEvent(incident)
	}
	
	// Check that events were recorded
	events := provider.GetEvents()
	assert.Len(t, events, 5)
	
	// Clear events
	provider.ClearEvents()
	
	// Check that events were cleared
	events = provider.GetEvents()
	assert.Empty(t, events)
}

// TestGetEventsByType tests filtering events by type
func TestGetEventsByType(t *testing.T) {
	provider := watchdog.NewDiagnosticsProvider()
	
	// Create resource usage for incidents
	resourceUsage := watchdog.ResourceUsage{
		CPUPercent:  90.0,
		MemoryMB:    500.0,
		Goroutines:  100,
		FileHandles: 50,
		GCPercent:   5.0,
		Timestamp:   time.Now(),
	}
	
	// Create and emit multiple incidents with different types
	incidentTypes := []watchdog.IncidentType{
		watchdog.IncidentResourceExceeded,
		watchdog.IncidentDeadlockDetected,
		watchdog.IncidentResourceExceeded,
		watchdog.IncidentCrash,
		watchdog.IncidentResourceExceeded,
	}
	
	for i, incidentType := range incidentTypes {
		incident := watchdog.Incident{
			ID:            fmt.Sprintf("test-incident-%d", i+1),
			Timestamp:     time.Now(),
			Type:          incidentType,
			Description:   fmt.Sprintf("Incident of type %s", incidentType),
			ResourceUsage: resourceUsage,
		}
		
		provider.EmitAgentDiagEvent(incident)
	}
	
	// Get events by type
	resourceExceededEvents := provider.GetEventsByType(string(watchdog.IncidentResourceExceeded))
	deadlockEvents := provider.GetEventsByType(string(watchdog.IncidentDeadlockDetected))
	crashEvents := provider.GetEventsByType(string(watchdog.IncidentCrash))
	restartFailedEvents := provider.GetEventsByType(string(watchdog.IncidentRestartFailed))
	
	// Check counts
	assert.Len(t, resourceExceededEvents, 3)
	assert.Len(t, deadlockEvents, 1)
	assert.Len(t, crashEvents, 1)
	assert.Len(t, restartFailedEvents, 0)
}

// TestGetEventsByComponent tests filtering events by component
func TestGetEventsByComponent(t *testing.T) {
	provider := watchdog.NewDiagnosticsProvider()
	
	// Create resource usage for incidents
	resourceUsage := watchdog.ResourceUsage{
		CPUPercent:  90.0,
		MemoryMB:    500.0,
		Goroutines:  100,
		FileHandles: 50,
		GCPercent:   5.0,
		Timestamp:   time.Now(),
	}
	
	// Create and emit events for different components
	components := []string{"component1", "component2", "component1", "component3", "component1"}
	
	for i, component := range components {
		incident := watchdog.Incident{
			ID:            fmt.Sprintf("test-incident-%d", i+1),
			Timestamp:     time.Now(),
			Type:          watchdog.IncidentResourceExceeded,
			Description:   fmt.Sprintf("Incident for %s", component),
			ResourceUsage: resourceUsage,
		}
		
		// Emit the event
		provider.EmitAgentDiagEvent(incident)
		
		// Set the component name (since we're testing the implementation)
		events := provider.GetEvents()
		lastEvent := &events[len(events)-1]
		lastEvent.ComponentName = component
	}
	
	// Get events by component
	component1Events := provider.GetEventsByComponent("component1")
	component2Events := provider.GetEventsByComponent("component2")
	component3Events := provider.GetEventsByComponent("component3")
	component4Events := provider.GetEventsByComponent("component4")
	
	// Check counts
	assert.Len(t, component1Events, 3)
	assert.Len(t, component2Events, 1)
	assert.Len(t, component3Events, 1)
	assert.Len(t, component4Events, 0)
}
