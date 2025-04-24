package collector

import (
	"context"
	"fmt"
	"regexp"
	"runtime"
	"sync"
	"time"
	
	"github.com/newrelic/infrastructure-agent/collector/platform"
)

// Register the process scanner at package initialization
func init() {
	RegisterCollector("process_scanner", func() Collector {
		return NewProcessScanner(DefaultConfig().ProcessScanner)
	})
}

// ProcessScanner implements a collector for process information
type ProcessScanner struct {
	config        ProcessScannerConfig
	platformCollector platform.ProcessCollector
	processCache  map[int]*ProcessInfo
	lastScanTime  time.Time
	metrics       *MetricsTracker
	registry      *ConsumerRegistry
	excludeRegexps []*regexp.Regexp
	includeRegexps []*regexp.Regexp
	ctx           context.Context
	cancel        context.CancelFunc
	scannerMutex  sync.RWMutex
	cacheMutex    sync.RWMutex
	scanTicker    *time.Ticker
	status        Status
	eventChannel  chan ProcessEvent
	wg            sync.WaitGroup
}

// NewProcessScanner creates a new process scanner
func NewProcessScanner(config ProcessScannerConfig) *ProcessScanner {
	return &ProcessScanner{
		config:       config,
		processCache: make(map[int]*ProcessInfo),
		metrics:      NewMetricsTracker(),
		registry:     NewConsumerRegistry(),
		status:       StatusInitialized,
		eventChannel: make(chan ProcessEvent, config.EventChannelSize),
	}
}

// Init initializes the process scanner
func (p *ProcessScanner) Init(ctx context.Context) error {
	p.scannerMutex.Lock()
	defer p.scannerMutex.Unlock()
	
	if p.status != StatusInitialized {
		return fmt.Errorf("scanner already initialized")
	}
	
	// Create a derived context
	p.ctx, p.cancel = context.WithCancel(ctx)
	
	// Create platform-specific collector
	options := map[string]interface{}{
		"procFSPath": p.config.ProcFSPath,
	}
	
	var err error
	p.platformCollector, err = platform.New(options)
	if err != nil {
		return fmt.Errorf("failed to create platform collector: %w", err)
	}
	
	// Compile exclude patterns
	p.excludeRegexps = make([]*regexp.Regexp, 0, len(p.config.ExcludePatterns))
	for _, pattern := range p.config.ExcludePatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid exclude pattern '%s': %w", pattern, err)
		}
		p.excludeRegexps = append(p.excludeRegexps, re)
	}
	
	// Compile include patterns
	p.includeRegexps = make([]*regexp.Regexp, 0, len(p.config.IncludePatterns))
	for _, pattern := range p.config.IncludePatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid include pattern '%s': %w", pattern, err)
		}
		p.includeRegexps = append(p.includeRegexps, re)
	}
	
	return nil
}

// Start begins the process scanning
func (p *ProcessScanner) Start() error {
	p.scannerMutex.Lock()
	defer p.scannerMutex.Unlock()
	
	if p.status == StatusRunning {
		return fmt.Errorf("scanner already running")
	}
	
	if p.status != StatusInitialized && p.status != StatusStopped && p.status != StatusPaused {
		return fmt.Errorf("scanner in invalid state: %s", p.status)
	}
	
	// Start the event processor
	p.wg.Add(1)
	go p.processEvents()
	
	// Start the scan ticker
	p.scanTicker = time.NewTicker(p.config.ScanInterval)
	p.wg.Add(1)
	go p.scanLoop()
	
	// Update status
	p.status = StatusRunning
	
	return nil
}

// Stop halts the process scanning
func (p *ProcessScanner) Stop() error {
	p.scannerMutex.Lock()
	defer p.scannerMutex.Unlock()
	
	if p.status != StatusRunning {
		return fmt.Errorf("scanner not running")
	}
	
	// Stop the ticker
	if p.scanTicker != nil {
		p.scanTicker.Stop()
	}
	
	// Cancel the context to signal all goroutines
	if p.cancel != nil {
		p.cancel()
	}
	
	// Wait for all goroutines to finish
	p.wg.Wait()
	
	// Update status
	p.status = StatusStopped
	
	return nil
}

// Status returns the current status of the scanner
func (p *ProcessScanner) Status() Status {
	p.scannerMutex.RLock()
	defer p.scannerMutex.RUnlock()
	
	return p.status
}

// Metrics returns performance metrics for the scanner
func (p *ProcessScanner) Metrics() map[string]float64 {
	return p.metrics.GetAllMetrics()
}

// Resources returns resource usage of the scanner itself
func (p *ProcessScanner) Resources() map[string]float64 {
	cpuPct, memBytes, err := p.platformCollector.GetSelfUsage()
	if err != nil {
		cpuPct, memBytes = 0, 0
	}
	
	return map[string]float64{
		"cpu_percent":  cpuPct,
		"memory_bytes": float64(memBytes),
	}
}

// Shutdown gracefully shuts down the scanner
func (p *ProcessScanner) Shutdown() error {
	// Stop scanning
	err := p.Stop()
	if err != nil && p.status != StatusStopped {
		return err
	}
	
	// Clean up resources
	if p.platformCollector != nil {
		err = p.platformCollector.Shutdown()
		if err != nil {
			return fmt.Errorf("error shutting down platform collector: %w", err)
		}
	}
	
	// Clear process cache
	p.cacheMutex.Lock()
	p.processCache = make(map[int]*ProcessInfo)
	p.cacheMutex.Unlock()
	
	return nil
}

// RegisterConsumer registers a consumer to receive process events
func (p *ProcessScanner) RegisterConsumer(name string, consumer ProcessConsumer) error {
	return p.registry.Register(name, consumer)
}

// UnregisterConsumer removes a registered consumer
func (p *ProcessScanner) UnregisterConsumer(name string) error {
	return p.registry.Unregister(name)
}

// scanLoop is the main scanning loop
func (p *ProcessScanner) scanLoop() {
	defer p.wg.Done()
	
	// Perform an initial scan
	p.performScan()
	
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-p.scanTicker.C:
			p.performScan()
		}
	}
}

// performScan executes a single scan cycle
func (p *ProcessScanner) performScan() {
	// Record metrics for scan duration
	stopTimer := p.metrics.StartTimer(MetricScanDuration)
	scanStart := time.Now()
	
	// Get current processes
	processes, err := p.platformCollector.GetProcesses()
	if err != nil {
		p.metrics.IncrementCounter(MetricScanErrors, 1)
		fmt.Printf("AgentDiagEvent: Error scanning processes: %v\n", err)
		return
	}
	
	// Apply filters
	filteredProcesses := p.filterProcesses(processes)
	
	// Update CPU times if needed
	if p.config.RefreshCPUStats {
		err = p.platformCollector.GetCPUTimes()
		if err != nil {
			p.metrics.IncrementCounter(MetricScanErrors, 1)
			fmt.Printf("AgentDiagEvent: Error refreshing CPU times: %v\n", err)
		}
	}
	
	// Process the filtered list
	processCount, created, updated, terminated := p.processNewScan(filteredProcesses)
	
	// Update metrics
	p.metrics.SetGauge(MetricProcessCount, float64(processCount))
	p.metrics.IncrementCounter(MetricProcessCreated, int64(created))
	p.metrics.IncrementCounter(MetricProcessUpdated, int64(updated))
	p.metrics.IncrementCounter(MetricProcessTerminated, int64(terminated))
	
	// Check for resource limits
	cpuPct, memBytes, _ := p.platformCollector.GetSelfUsage()
	p.metrics.SetGauge(MetricCPUUsage, cpuPct)
	p.metrics.SetGauge(MetricMemoryUsage, float64(memBytes))
	
	if cpuPct > p.config.MaxCPUUsage {
		p.metrics.IncrementCounter(MetricLimitBreaches, 1)
		fmt.Printf("AgentDiagEvent: ModuleOverLimit detected in process scanner. CPU: %.2f%% (limit: %.2f%%)\n",
			cpuPct, p.config.MaxCPUUsage)
		
		// Adjust scan interval if adaptive sampling is enabled
		if p.config.AdaptiveSampling {
			p.adjustScanInterval(cpuPct)
		}
	}
	
	// Stop the timer and record scan duration
	stopTimer()
	scanDuration := time.Since(scanStart)
	
	// Set metrics for scan interval
	p.metrics.SetGauge(MetricEventQueueSize, float64(len(p.eventChannel)))
	p.metrics.SetGauge(MetricConsumerCount, float64(p.registry.ConsumerCount()))
	
	// Record when we did the scan
	p.lastScanTime = time.Now()
	
	// Check if scan took too long
	if scanDuration > p.config.MaxScanTime {
		fmt.Printf("AgentDiagEvent: Scan duration exceeded limit: %v (limit: %v)\n",
			scanDuration, p.config.MaxScanTime)
	}
}

// filterProcesses applies include/exclude filters to the process list
func (p *ProcessScanner) filterProcesses(processes []*ProcessInfo) []*ProcessInfo {
	if len(p.includeRegexps) == 0 && len(p.excludeRegexps) == 0 {
		return processes
	}
	
	var filtered []*ProcessInfo
	
	for _, proc := range processes {
		// Apply exclude patterns first
		excluded := false
		for _, re := range p.excludeRegexps {
			if re.MatchString(proc.Command) || re.MatchString(proc.Name) {
				excluded = true
				break
			}
		}
		
		if excluded {
			continue
		}
		
		// If include patterns exist, process must match at least one
		if len(p.includeRegexps) > 0 {
			included := false
			for _, re := range p.includeRegexps {
				if re.MatchString(proc.Command) || re.MatchString(proc.Name) {
					included = true
					break
				}
			}
			
			if !included {
				continue
			}
		}
		
		filtered = append(filtered, proc)
	}
	
	return filtered
}

// processNewScan compares new process list with cached processes to detect events
func (p *ProcessScanner) processNewScan(newProcesses []*ProcessInfo) (int, int, int, int) {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()
	
	// Create a map of new processes for quick lookup
	newProcessMap := make(map[int]*ProcessInfo, len(newProcesses))
	for _, proc := range newProcesses {
		newProcessMap[proc.PID] = proc
	}
	
	created := 0
	updated := 0
	terminated := 0
	
	// Check for terminated processes
	for pid, cachedProc := range p.processCache {
		if _, exists := newProcessMap[pid]; !exists {
			// Process no longer exists
			terminated++
			delete(p.processCache, pid)
			
			// Generate terminated event
			p.queueEvent(ProcessEvent{
				Type:      ProcessTerminated,
				Process:   cachedProc.Clone(),
				Timestamp: time.Now(),
			})
		}
	}
	
	// Check for new and updated processes
	for pid, newProc := range newProcessMap {
		cachedProc, exists := p.processCache[pid]
		
		if !exists {
			// New process
			created++
			p.processCache[pid] = newProc.Clone()
			
			// Generate created event
			p.queueEvent(ProcessEvent{
				Type:      ProcessCreated,
				Process:   newProc.Clone(),
				Timestamp: time.Now(),
			})
		} else {
			// Existing process, check if it has changed
			if !cachedProc.Equal(newProc) {
				updated++
				p.processCache[pid] = newProc.Clone()
				
				// Generate updated event
				p.queueEvent(ProcessEvent{
					Type:      ProcessUpdated,
					Process:   newProc.Clone(),
					Timestamp: time.Now(),
				})
			}
		}
	}
	
	return len(p.processCache), created, updated, terminated
}

// queueEvent adds an event to the event channel
func (p *ProcessScanner) queueEvent(event ProcessEvent) {
	// Non-blocking send to event channel with timeout
	select {
	case p.eventChannel <- event:
		// Event queued successfully
	case <-time.After(100 * time.Millisecond):
		// Channel is full or blocked
		p.metrics.IncrementCounter(MetricNotificationErrors, 1)
		fmt.Printf("AgentDiagEvent: Event channel full, dropping event for PID %d\n", event.Process.PID)
	}
}

// processEvents handles events from the event channel
func (p *ProcessScanner) processEvents() {
	defer p.wg.Done()
	
	batchSize := p.config.EventBatchSize
	if batchSize <= 0 {
		batchSize = 100
	}
	
	for {
		select {
		case <-p.ctx.Done():
			return
		case event := <-p.eventChannel:
			// Process the event
			errors := p.registry.NotifyAll(event)
			if len(errors) > 0 {
				p.metrics.IncrementCounter(MetricNotificationErrors, int64(len(errors)))
				for _, err := range errors {
					fmt.Printf("AgentDiagEvent: Error notifying consumers: %v\n", err)
				}
			}
		}
	}
}

// adjustScanInterval modifies the scan interval based on CPU usage
func (p *ProcessScanner) adjustScanInterval(cpuPct float64) {
	p.scannerMutex.Lock()
	defer p.scannerMutex.Unlock()
	
	if !p.config.AdaptiveSampling || p.scanTicker == nil {
		return
	}
	
	currentInterval := p.config.ScanInterval
	
	// Calculate a new interval based on how much we're exceeding the target
	ratio := cpuPct / p.config.MaxCPUUsage
	
	// Only adjust if we're significantly over or under
	if ratio > 1.2 {
		// CPU usage too high, increase interval (slow down)
		newInterval := time.Duration(float64(currentInterval) * (ratio * 1.2))
		
		// Cap at a reasonable maximum (e.g., 1 minute)
		if newInterval > time.Minute {
			newInterval = time.Minute
		}
		
		if newInterval != currentInterval {
			p.metrics.IncrementCounter(MetricAdaptiveRateChanges, 1)
			fmt.Printf("AgentDiagEvent: Increasing scan interval from %v to %v due to high CPU usage (%.2f%%)\n",
				currentInterval, newInterval, cpuPct)
			
			p.scanTicker.Reset(newInterval)
			p.config.ScanInterval = newInterval
		}
	} else if ratio < 0.5 && currentInterval > time.Second*10 {
		// CPU usage well below target and current interval is longer than default,
		// decrease interval (speed up) to approach target
		newInterval := time.Duration(float64(currentInterval) * 0.8)
		
		// Don't go below the original configured interval
		if newInterval < time.Second*10 {
			newInterval = time.Second * 10
		}
		
		if newInterval != currentInterval {
			p.metrics.IncrementCounter(MetricAdaptiveRateChanges, 1)
			fmt.Printf("AgentDiagEvent: Decreasing scan interval from %v to %v due to low CPU usage (%.2f%%)\n",
				currentInterval, newInterval, cpuPct)
			
			p.scanTicker.Reset(newInterval)
			p.config.ScanInterval = newInterval
		}
	}
}

// ForceScan triggers an immediate scan
func (p *ProcessScanner) ForceScan() error {
	if p.status != StatusRunning {
		return fmt.Errorf("scanner not running")
	}
	
	go p.performScan()
	return nil
}

// GetCachedProcesses returns a copy of the current process cache
func (p *ProcessScanner) GetCachedProcesses() []*ProcessInfo {
	p.cacheMutex.RLock()
	defer p.cacheMutex.RUnlock()
	
	processes := make([]*ProcessInfo, 0, len(p.processCache))
	for _, proc := range p.processCache {
		processes = append(processes, proc.Clone())
	}
	
	return processes
}

// GetCachedProcess returns a specific process from the cache
func (p *ProcessScanner) GetCachedProcess(pid int) (*ProcessInfo, bool) {
	p.cacheMutex.RLock()
	defer p.cacheMutex.RUnlock()
	
	proc, exists := p.processCache[pid]
	if !exists {
		return nil, false
	}
	
	return proc.Clone(), true
}
