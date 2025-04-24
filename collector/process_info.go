package collector

import (
	"fmt"
	"time"
)

// ProcessInfo represents detailed information about a process
type ProcessInfo struct {
	// PID is the process identifier
	PID int `json:"pid"`
	
	// PPID is the parent process identifier
	PPID int `json:"ppid"`
	
	// Name is the process name
	Name string `json:"name"`
	
	// Executable is the path to the executable
	Executable string `json:"executable"`
	
	// Command is the command line with arguments
	Command string `json:"command"`
	
	// User is the username of the process owner
	User string `json:"user"`
	
	// CPU is the percentage of CPU usage (0-100)
	CPU float64 `json:"cpu"`
	
	// RSS is the resident set size in bytes
	RSS int64 `json:"rss"`
	
	// VMS is the virtual memory size in bytes
	VMS int64 `json:"vms"`
	
	// FDs is the number of open file descriptors
	FDs int `json:"fds"`
	
	// Threads is the number of threads
	Threads int `json:"threads"`
	
	// StartTime is when the process started
	StartTime time.Time `json:"startTime"`
	
	// State is the process state
	State string `json:"state"`
	
	// LastUpdated is when this information was last updated
	LastUpdated time.Time `json:"lastUpdated"`
	
	// IOReadBytes is the total bytes read from disk
	IOReadBytes int64 `json:"ioReadBytes"`
	
	// IOWriteBytes is the total bytes written to disk
	IOWriteBytes int64 `json:"ioWriteBytes"`
	
	// Labels are optional key-value pairs for additional information
	Labels map[string]string `json:"labels,omitempty"`
}

// DeltaProcessInfo represents changes in process metrics between two samples
type DeltaProcessInfo struct {
	// PID of the process
	PID int `json:"pid"`
	
	// DeltaTime is the time between samples
	DeltaTime time.Duration `json:"deltaTime"`
	
	// CPU is the delta in CPU usage
	CPU float64 `json:"cpu"`
	
	// RSS is the delta in resident set size
	RSS int64 `json:"rss"`
	
	// IOReadBytes is the delta in bytes read from disk
	IOReadBytes int64 `json:"ioReadBytes"`
	
	// IOWriteBytes is the delta in bytes written to disk
	IOWriteBytes int64 `json:"ioWriteBytes"`
}

// CalculateDelta computes the differences between two process info snapshots
func CalculateDelta(current, previous *ProcessInfo) (*DeltaProcessInfo, error) {
	if current == nil || previous == nil {
		return nil, fmt.Errorf("both current and previous process info must be non-nil")
	}
	
	if current.PID != previous.PID {
		return nil, fmt.Errorf("cannot calculate delta for different processes")
	}
	
	deltaTime := current.LastUpdated.Sub(previous.LastUpdated)
	if deltaTime <= 0 {
		return nil, fmt.Errorf("invalid time delta: %v", deltaTime)
	}
	
	return &DeltaProcessInfo{
		PID:         current.PID,
		DeltaTime:   deltaTime,
		CPU:         current.CPU - previous.CPU,
		RSS:         current.RSS - previous.RSS,
		IOReadBytes: current.IOReadBytes - previous.IOReadBytes,
		IOWriteBytes: current.IOWriteBytes - previous.IOWriteBytes,
	}, nil
}

// Clone creates a deep copy of ProcessInfo
func (p *ProcessInfo) Clone() *ProcessInfo {
	if p == nil {
		return nil
	}
	
	newLabels := make(map[string]string, len(p.Labels))
	for k, v := range p.Labels {
		newLabels[k] = v
	}
	
	return &ProcessInfo{
		PID:         p.PID,
		PPID:        p.PPID,
		Name:        p.Name,
		Executable:  p.Executable,
		Command:     p.Command,
		User:        p.User,
		CPU:         p.CPU,
		RSS:         p.RSS,
		VMS:         p.VMS,
		FDs:         p.FDs,
		Threads:     p.Threads,
		StartTime:   p.StartTime,
		State:       p.State,
		LastUpdated: p.LastUpdated,
		IOReadBytes: p.IOReadBytes,
		IOWriteBytes: p.IOWriteBytes,
		Labels:      newLabels,
	}
}

// Equal checks if two ProcessInfo instances are equal
func (p *ProcessInfo) Equal(other *ProcessInfo) bool {
	if p == nil && other == nil {
		return true
	}
	
	if p == nil || other == nil {
		return false
	}
	
	// Check basic fields
	if p.PID != other.PID ||
		p.PPID != other.PPID ||
		p.Name != other.Name ||
		p.Executable != other.Executable ||
		p.Command != other.Command ||
		p.User != other.User ||
		p.CPU != other.CPU ||
		p.RSS != other.RSS ||
		p.VMS != other.VMS ||
		p.FDs != other.FDs ||
		p.Threads != other.Threads ||
		p.State != other.State ||
		p.IOReadBytes != other.IOReadBytes ||
		p.IOWriteBytes != other.IOWriteBytes ||
		!p.StartTime.Equal(other.StartTime) {
		return false
	}
	
	// Check labels
	if len(p.Labels) != len(other.Labels) {
		return false
	}
	
	for k, v := range p.Labels {
		if otherVal, ok := other.Labels[k]; !ok || v != otherVal {
			return false
		}
	}
	
	return true
}

// GetKey returns a unique identifier for the process
func (p *ProcessInfo) GetKey() string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%d-%s", p.PID, p.StartTime.Format("20060102150405"))
}

// ProcessSummary returns a compact string representation of the process
func (p *ProcessInfo) ProcessSummary() string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprintf("PID=%d Name=%s CPU=%.1f%% RSS=%d MB", 
		p.PID, p.Name, p.CPU, p.RSS/(1024*1024))
}
