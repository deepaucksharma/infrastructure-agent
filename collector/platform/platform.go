// Package platform provides platform-specific implementations for process data collection.
package platform

import (
	"fmt"
	"runtime"
	"time"
	
	"github.com/newrelic/infrastructure-agent/collector"
)

// ProcessCollector defines the interface for platform-specific process collection
type ProcessCollector interface {
	// GetProcesses returns a list of all processes on the system
	GetProcesses() ([]*collector.ProcessInfo, error)
	
	// GetProcess returns detailed information about a specific process
	GetProcess(pid int) (*collector.ProcessInfo, error)
	
	// IsProcessRunning checks if a process is running
	IsProcessRunning(pid int) bool
	
	// GetProcessCount returns the total number of processes on the system
	GetProcessCount() (int, error)
	
	// GetCPUTimes returns CPU times for the system and processes
	GetCPUTimes() error
	
	// GetMemoryStats returns memory information for the system
	GetMemoryStats() (uint64, uint64, error) // total, used, error
	
	// GetSelfUsage returns the resource usage of the current process
	GetSelfUsage() (float64, uint64, error) // cpu%, memory bytes, error
	
	// Shutdown cleans up any resources
	Shutdown() error
}

// New creates a new platform-specific process collector
func New(options map[string]interface{}) (ProcessCollector, error) {
	switch runtime.GOOS {
	case "linux":
		return NewLinuxProcessCollector(options)
	case "windows":
		return NewWindowsProcessCollector(options)
	case "darwin":
		return NewDarwinProcessCollector(options)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// LinuxProcessCollector collects process information on Linux
type LinuxProcessCollector struct {
	procFSPath    string
	lastCPUTimes  map[int]time.Time
	systemCPUTime float64
	lastUpdateTime time.Time
}

// NewLinuxProcessCollector creates a new Linux process collector
func NewLinuxProcessCollector(options map[string]interface{}) (*LinuxProcessCollector, error) {
	procFSPath := "/proc"
	if path, ok := options["procFSPath"].(string); ok && path != "" {
		procFSPath = path
	}
	
	return &LinuxProcessCollector{
		procFSPath:   procFSPath,
		lastCPUTimes: make(map[int]time.Time),
		lastUpdateTime: time.Now(),
	}, nil
}

// GetProcesses returns a list of all processes on Linux
func (l *LinuxProcessCollector) GetProcesses() ([]*collector.ProcessInfo, error) {
	// In a real implementation, this would:
	// 1. Read /proc directory
	// 2. Parse each numeric directory (PID)
	// 3. Extract process information from /proc/[pid]/stat, /proc/[pid]/status, etc.
	
	// Placeholder implementation
	return []*collector.ProcessInfo{
		{
			PID:         1,
			PPID:        0,
			Name:        "systemd",
			Executable:  "/usr/lib/systemd/systemd",
			Command:     "/usr/lib/systemd/systemd --system --deserialize=31",
			User:        "root",
			CPU:         0.1,
			RSS:         4 * 1024 * 1024,
			VMS:         120 * 1024 * 1024,
			FDs:         64,
			Threads:     1,
			StartTime:   time.Now().Add(-24 * time.Hour),
			State:       "S",
			LastUpdated: time.Now(),
		},
		{
			PID:         100,
			PPID:        1,
			Name:        "sshd",
			Executable:  "/usr/sbin/sshd",
			Command:     "/usr/sbin/sshd -D",
			User:        "root",
			CPU:         0.2,
			RSS:         2 * 1024 * 1024,
			VMS:         60 * 1024 * 1024,
			FDs:         32,
			Threads:     1,
			StartTime:   time.Now().Add(-12 * time.Hour),
			State:       "S",
			LastUpdated: time.Now(),
		},
	}, nil
}

// GetProcess returns detailed information about a specific process on Linux
func (l *LinuxProcessCollector) GetProcess(pid int) (*collector.ProcessInfo, error) {
	// Placeholder implementation
	return &collector.ProcessInfo{
		PID:         pid,
		PPID:        1,
		Name:        fmt.Sprintf("process-%d", pid),
		Executable:  fmt.Sprintf("/usr/bin/process-%d", pid),
		Command:     fmt.Sprintf("/usr/bin/process-%d --arg=value", pid),
		User:        "root",
		CPU:         0.1,
		RSS:         2 * 1024 * 1024,
		VMS:         60 * 1024 * 1024,
		FDs:         32,
		Threads:     1,
		StartTime:   time.Now().Add(-1 * time.Hour),
		State:       "S",
		LastUpdated: time.Now(),
	}, nil
}

// IsProcessRunning checks if a process is running on Linux
func (l *LinuxProcessCollector) IsProcessRunning(pid int) bool {
	// Placeholder implementation
	return pid > 0 && pid < 32768
}

// GetProcessCount returns the total number of processes on Linux
func (l *LinuxProcessCollector) GetProcessCount() (int, error) {
	// Placeholder implementation
	return 100, nil
}

// GetCPUTimes updates CPU times for processes on Linux
func (l *LinuxProcessCollector) GetCPUTimes() error {
	// Placeholder implementation
	l.lastUpdateTime = time.Now()
	return nil
}

// GetMemoryStats returns memory information for the Linux system
func (l *LinuxProcessCollector) GetMemoryStats() (uint64, uint64, error) {
	// Placeholder implementation
	return 8 * 1024 * 1024 * 1024, 4 * 1024 * 1024 * 1024, nil
}

// GetSelfUsage returns the resource usage of the current process
func (l *LinuxProcessCollector) GetSelfUsage() (float64, uint64, error) {
	// Placeholder implementation
	return 0.2, 50 * 1024 * 1024, nil
}

// Shutdown cleans up any resources
func (l *LinuxProcessCollector) Shutdown() error {
	return nil
}

// WindowsProcessCollector collects process information on Windows
type WindowsProcessCollector struct {
	lastCPUTimes  map[int]time.Time
	systemCPUTime float64
	lastUpdateTime time.Time
}

// NewWindowsProcessCollector creates a new Windows process collector
func NewWindowsProcessCollector(options map[string]interface{}) (*WindowsProcessCollector, error) {
	return &WindowsProcessCollector{
		lastCPUTimes: make(map[int]time.Time),
		lastUpdateTime: time.Now(),
	}, nil
}

// GetProcesses returns a list of all processes on Windows
func (w *WindowsProcessCollector) GetProcesses() ([]*collector.ProcessInfo, error) {
	// Placeholder implementation
	return []*collector.ProcessInfo{
		{
			PID:         4,
			PPID:        0,
			Name:        "System",
			Executable:  "",
			Command:     "",
			User:        "SYSTEM",
			CPU:         0.1,
			RSS:         4 * 1024 * 1024,
			VMS:         120 * 1024 * 1024,
			FDs:         0,
			Threads:     100,
			StartTime:   time.Now().Add(-24 * time.Hour),
			State:       "Running",
			LastUpdated: time.Now(),
		},
		{
			PID:         400,
			PPID:        4,
			Name:        "svchost.exe",
			Executable:  "C:\\Windows\\System32\\svchost.exe",
			Command:     "C:\\Windows\\System32\\svchost.exe -k LocalService",
			User:        "SYSTEM",
			CPU:         0.2,
			RSS:         8 * 1024 * 1024,
			VMS:         80 * 1024 * 1024,
			FDs:         0,
			Threads:     10,
			StartTime:   time.Now().Add(-12 * time.Hour),
			State:       "Running",
			LastUpdated: time.Now(),
		},
	}, nil
}

// GetProcess returns detailed information about a specific process on Windows
func (w *WindowsProcessCollector) GetProcess(pid int) (*collector.ProcessInfo, error) {
	// Placeholder implementation
	return &collector.ProcessInfo{
		PID:         pid,
		PPID:        4,
		Name:        fmt.Sprintf("process-%d.exe", pid),
		Executable:  fmt.Sprintf("C:\\Program Files\\Process\\process-%d.exe", pid),
		Command:     fmt.Sprintf("C:\\Program Files\\Process\\process-%d.exe --arg=value", pid),
		User:        "Administrator",
		CPU:         0.1,
		RSS:         4 * 1024 * 1024,
		VMS:         60 * 1024 * 1024,
		FDs:         0,
		Threads:     2,
		StartTime:   time.Now().Add(-1 * time.Hour),
		State:       "Running",
		LastUpdated: time.Now(),
	}, nil
}

// IsProcessRunning checks if a process is running on Windows
func (w *WindowsProcessCollector) IsProcessRunning(pid int) bool {
	// Placeholder implementation
	return pid > 0 && pid < 32768
}

// GetProcessCount returns the total number of processes on Windows
func (w *WindowsProcessCollector) GetProcessCount() (int, error) {
	// Placeholder implementation
	return 120, nil
}

// GetCPUTimes updates CPU times for processes on Windows
func (w *WindowsProcessCollector) GetCPUTimes() error {
	// Placeholder implementation
	w.lastUpdateTime = time.Now()
	return nil
}

// GetMemoryStats returns memory information for the Windows system
func (w *WindowsProcessCollector) GetMemoryStats() (uint64, uint64, error) {
	// Placeholder implementation
	return 16 * 1024 * 1024 * 1024, 8 * 1024 * 1024 * 1024, nil
}

// GetSelfUsage returns the resource usage of the current process
func (w *WindowsProcessCollector) GetSelfUsage() (float64, uint64, error) {
	// Placeholder implementation
	return 0.3, 60 * 1024 * 1024, nil
}

// Shutdown cleans up any resources
func (w *WindowsProcessCollector) Shutdown() error {
	return nil
}

// DarwinProcessCollector collects process information on macOS
type DarwinProcessCollector struct {
	lastCPUTimes  map[int]time.Time
	systemCPUTime float64
	lastUpdateTime time.Time
}

// NewDarwinProcessCollector creates a new macOS process collector
func NewDarwinProcessCollector(options map[string]interface{}) (*DarwinProcessCollector, error) {
	return &DarwinProcessCollector{
		lastCPUTimes: make(map[int]time.Time),
		lastUpdateTime: time.Now(),
	}, nil
}

// GetProcesses returns a list of all processes on macOS
func (d *DarwinProcessCollector) GetProcesses() ([]*collector.ProcessInfo, error) {
	// Placeholder implementation
	return []*collector.ProcessInfo{
		{
			PID:         1,
			PPID:        0,
			Name:        "launchd",
			Executable:  "/sbin/launchd",
			Command:     "/sbin/launchd",
			User:        "root",
			CPU:         0.1,
			RSS:         4 * 1024 * 1024,
			VMS:         120 * 1024 * 1024,
			FDs:         100,
			Threads:     5,
			StartTime:   time.Now().Add(-24 * time.Hour),
			State:       "S",
			LastUpdated: time.Now(),
		},
		{
			PID:         200,
			PPID:        1,
			Name:        "Finder",
			Executable:  "/System/Library/CoreServices/Finder.app/Contents/MacOS/Finder",
			Command:     "/System/Library/CoreServices/Finder.app/Contents/MacOS/Finder",
			User:        "user",
			CPU:         0.5,
			RSS:         80 * 1024 * 1024,
			VMS:         200 * 1024 * 1024,
			FDs:         50,
			Threads:     8,
			StartTime:   time.Now().Add(-12 * time.Hour),
			State:       "S",
			LastUpdated: time.Now(),
		},
	}, nil
}

// GetProcess returns detailed information about a specific process on macOS
func (d *DarwinProcessCollector) GetProcess(pid int) (*collector.ProcessInfo, error) {
	// Placeholder implementation
	return &collector.ProcessInfo{
		PID:         pid,
		PPID:        1,
		Name:        fmt.Sprintf("process-%d", pid),
		Executable:  fmt.Sprintf("/Applications/Process-%d.app/Contents/MacOS/Process-%d", pid, pid),
		Command:     fmt.Sprintf("/Applications/Process-%d.app/Contents/MacOS/Process-%d --arg=value", pid, pid),
		User:        "user",
		CPU:         0.2,
		RSS:         40 * 1024 * 1024,
		VMS:         120 * 1024 * 1024,
		FDs:         30,
		Threads:     3,
		StartTime:   time.Now().Add(-1 * time.Hour),
		State:       "S",
		LastUpdated: time.Now(),
	}, nil
}

// IsProcessRunning checks if a process is running on macOS
func (d *DarwinProcessCollector) IsProcessRunning(pid int) bool {
	// Placeholder implementation
	return pid > 0 && pid < 32768
}

// GetProcessCount returns the total number of processes on macOS
func (d *DarwinProcessCollector) GetProcessCount() (int, error) {
	// Placeholder implementation
	return 150, nil
}

// GetCPUTimes updates CPU times for processes on macOS
func (d *DarwinProcessCollector) GetCPUTimes() error {
	// Placeholder implementation
	d.lastUpdateTime = time.Now()
	return nil
}

// GetMemoryStats returns memory information for the macOS system
func (d *DarwinProcessCollector) GetMemoryStats() (uint64, uint64, error) {
	// Placeholder implementation
	return 16 * 1024 * 1024 * 1024, 8 * 1024 * 1024 * 1024, nil
}

// GetSelfUsage returns the resource usage of the current process
func (d *DarwinProcessCollector) GetSelfUsage() (float64, uint64, error) {
	// Placeholder implementation
	return 0.3, 60 * 1024 * 1024, nil
}

// Shutdown cleans up any resources
func (d *DarwinProcessCollector) Shutdown() error {
	return nil
}
