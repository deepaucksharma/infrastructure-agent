# Resilient Infrastructure-Agent Telemetry Foundation Runbook

## Introduction
This runbook provides operational documentation for the Infrastructure-Agent Telemetry Foundation. It covers configuration parameters, metrics, events, diagnostic procedures, and operational scenarios.

## Table of Contents
1. [Feature Overview](#feature-overview)
2. [Configuration Parameters](#configuration-parameters)
3. [Metrics and Events](#metrics-and-events)
4. [Diagnostic Procedures](#diagnostic-procedures)
5. [Operational Scenarios](#operational-scenarios)

## Feature Overview
The Infrastructure-Agent Telemetry Foundation provides high-fidelity process monitoring with statistical accuracy and host safety guarantees.

### Core Modules
- **Collector**: Gathers raw telemetry data from the host system
- **Sampler**: Implements efficient sampling strategies
- **Sketch**: Statistical data structures for accurate percentile calculations
- **Watchdog**: Self-governance and circuit breaker functionality

### Process Scanner
The Process Scanner efficiently collects information about running processes on the host system while maintaining low resource usage. It provides accurate and timely process data to other components and consumers while handling process lifecycle events.

Key features:
- Efficient process collection on multiple platforms (Linux, Windows, macOS)
- Event-based detection of process creation, updates, and termination
- Adaptive scanning rate based on system load
- Configurable inclusion/exclusion filters
- Thread-safe consumer notification system
- Self-monitoring for CPU and memory usage

### Top-N Process Sampler
The Top-N Process Sampler efficiently tracks the most resource-intensive processes on a system while maintaining a high capture ratio (≥95%) and low CPU overhead (≤0.5%). It uses a min-heap data structure to maintain the processes with the highest resource usage scores, enabling O(log N) operations for insertions, deletions, and updates.

Key features:
- Adaptive scoring based on weighted CPU and RSS metrics
- Circuit breaker pattern for self-protection under high load
- Optimized handling of high process churn (up to 2000 PIDs/s)
- Configurable process scoring weights and thresholds
- Comprehensive metrics for monitoring capture ratio and performance

### DDSketch Statistical Aggregation
The DDSketch (Distributed Distribution Sketch) module provides accurate and memory-efficient percentile calculations with guaranteed relative error bounds. It is designed for distributed environments and enables high-fidelity statistical analysis of telemetry data.

Key features:
- Guaranteed relative error bounds (≤1% for p95/p99 with γ=0.0075)
- Memory-efficient representation with sparse and dense store options
- Automatic store type switching based on data distribution
- Mergeable design for distributed aggregation
- Full serialization support for transport
- Thread-safe implementation for concurrent access

## Configuration Parameters
This section documents all configuration parameters for the agent.

### Global Configuration
| Parameter | Default | Description |
|-----------|---------|-------------|
| `logLevel` | `info` | Logging level (debug, info, warn, error) |
| `metricsInterval` | `15s` | Interval for metrics collection |

### Process Scanner Configuration
| Parameter | Default | Description |
|-----------|---------|-------------|
| `collectorType` | `process_scanner` | Type of collector implementation |
| `collectionInterval` | `15s` | General collection interval |
| `maxCPUUsage` | `0.75` | Maximum allowed CPU percentage for the collector |
| `processScanner.enabled` | `true` | Whether process scanning is enabled |
| `processScanner.scanInterval` | `10s` | How often to scan for processes |
| `processScanner.maxProcesses` | `3000` | Maximum number of processes to track |
| `processScanner.excludePatterns` | `[]` | Regex patterns for processes to exclude |
| `processScanner.includePatterns` | `[]` | Regex patterns for processes to include |
| `processScanner.procFSPath` | `/proc` | Path to procfs (Linux only) |
| `processScanner.refreshCPUStats` | `true` | Whether to refresh CPU statistics between scans |
| `processScanner.eventBatchSize` | `100` | Maximum number of events to send in one batch |
| `processScanner.eventChannelSize` | `1000` | Size of the event channel buffer |
| `processScanner.retryInterval` | `5s` | Time to wait before retrying after failure |
| `processScanner.adaptiveSampling` | `true` | Enables adaptive scanning based on system load |
| `processScanner.maxScanTime` | `200ms` | Maximum time allowed for a full scan |

### Top-N Sampler Configuration
| Parameter | Default | Description |
|-----------|---------|-------------|
| `samplerType` | `topn` | Type of sampler implementation |
| `sampleInterval` | `15s` | How often to sample processes |
| `maxSamplerCPU` | `0.5` | Maximum allowed CPU percentage for the sampler |
| `topN.maxProcesses` | `500` | Maximum number of processes to track |
| `topN.cpuWeight` | `0.7` | Weight given to CPU usage in scoring (0-1) |
| `topN.rssWeight` | `0.3` | Weight given to memory usage in scoring (0-1) |
| `topN.minScore` | `0.001` | Minimum score a process must have to be tracked |
| `topN.stabilityFactor` | `0.8` | Affects how quickly scores change (0-1) |
| `topN.churnHandlingEnabled` | `true` | Enables optimizations for high PID churn |
| `topN.churnThreshold` | `2000` | PID churn rate that activates optimizations (PIDs/s) |

### DDSketch Configuration
| Parameter | Default | Description |
|-----------|---------|-------------|
| `sketchType` | `ddsketch` | Type of sketch implementation |
| `ddSketch.relativeAccuracy` | `0.0075` | Gamma (γ) parameter controlling accuracy (0-1) |
| `ddSketch.minValue` | `1e-9` | Minimum value that can be stored in the sketch |
| `ddSketch.maxValue` | `1e9` | Maximum value that can be stored in the sketch |
| `ddSketch.initialCapacity` | `128` | Initial capacity for bucket stores |
| `ddSketch.useSparseStore` | `true` | Whether to use sparse store initially |
| `ddSketch.collapseThreshold` | `10` | Threshold for collapsing sparse buckets |
| `ddSketch.autoSwitch` | `true` | Enables automatic switching between store types |
| `ddSketch.switchThreshold` | `0.5` | Density threshold for switching store types (0-1) |

## Metrics and Events
This section documents all metrics and events emitted by the agent.

### Core Metrics
| Metric Name | Type | Description | Normal Range |
|-------------|------|-------------|--------------|
| `infra.cpu.usage` | Gauge | CPU usage percentage per process | 0-100% |
| `infra.memory.rss` | Gauge | Resident Set Size per process | Varies by process |

### Process Scanner Metrics
| Metric Name | Type | Description | Normal Range |
|-------------|------|-------------|--------------|
| `scan_duration_ms` | Gauge | Time taken to complete a process scan | 10-200ms |
| `cpu_usage_percent` | Gauge | CPU usage of the scanner itself | 0-0.75% |
| `memory_usage_bytes` | Gauge | Memory usage of the scanner itself | 0-30MB |
| `process_count` | Gauge | Number of processes being tracked | 50-3000 |
| `process_created_total` | Counter | Total number of process creation events | Increases over time |
| `process_updated_total` | Counter | Total number of process update events | Increases over time |
| `process_terminated_total` | Counter | Total number of process termination events | Increases over time |
| `scan_errors_total` | Counter | Total number of scan errors | Should remain low |
| `limit_breaches_total` | Counter | Total number of resource limit breaches | Should remain low |
| `notification_errors_total` | Counter | Total number of errors notifying consumers | Should remain low |
| `scan_interval_actual_ms` | Gauge | Actual interval between scans | Close to configured value |
| `adaptive_rate_changes_total` | Counter | Total number of adaptive rate changes | Increases during load |
| `event_queue_size` | Gauge | Current size of the event queue | Should not stay at max |
| `consumer_count` | Gauge | Number of registered consumers | Typically stable |

### Top-N Sampler Metrics
| Metric Name | Type | Description | Normal Range |
|-------------|------|-------------|--------------|
| `topn_capture_ratio` | Gauge | Percentage of total CPU captured by tracked processes | 95-100% |
| `topn_update_time_seconds` | Gauge | Time taken to update the sampler | 0-0.01s |
| `topn_processes_tracked` | Gauge | Number of processes being tracked | 0-500 |
| `topn_processes_updated` | Counter | Number of processes updated in last interval | Varies |
| `topn_churn_rate` | Gauge | Process creation/deletion rate (PIDs/s) | 0-2000 |
| `topn_circuit_breaker` | State | Whether circuit breaker is active (0=closed, 1=open) | 0 |
| `sampler_cpu_percent` | Gauge | CPU usage of the sampler itself | 0-0.5% |
| `sampler_rss_bytes` | Gauge | Memory usage of the sampler itself | 0-10MB |
| `sampler_uptime_seconds` | Counter | Uptime of the sampler | Increases steadily |

### DDSketch Metrics
| Metric Name | Type | Description | Normal Range |
|-------------|------|-------------|--------------|
| `sketch_count` | Gauge | Number of values in the sketch | Varies |
| `sketch_buckets` | Gauge | Number of buckets in the sketch | Varies |
| `sketch_memory_bytes` | Gauge | Memory usage of the sketch | 0-20MB |
| `sketch_store_density` | Gauge | Density of the sketch store (%) | 0-100% |
| `sketch_uptime_seconds` | Counter | Uptime of the sketch | Increases steadily |

### Agent Diagnostic Events
| Event Name | Description | Trigger Condition |
|------------|-------------|------------------|
| `AgentDiagEvent` | General diagnostic event | Various agent conditions |
| `ModuleOverLimit` | Module exceeded resource limits | CPU/Memory threshold breach |
| `StoreTypeSwitched` | DDSketch switched store types | Store density crossed threshold |
| `ScanRateAdjusted` | Process scan rate was adjusted | High CPU usage or low resources |
| `ProcessEventDropped` | Process event was dropped | Event queue overflow |

## Diagnostic Procedures
This section provides guidance for diagnosing and resolving common issues.

### Common Issues

#### High CPU Usage
**Symptoms:**
- Agent CPU usage consistently above 2%
- Host performance degradation

**Troubleshooting Steps:**
1. Check logs for `ModuleOverLimit` events
2. Review module configuration parameters
3. Check metrics for which module is consuming resources
4. Look for high process count or churn rate

**Resolution:**
- Increase CPU thresholds for specific modules
- Reduce scanning/sampling frequency
- Apply more restrictive process filters
- Enable adaptive sampling if not already enabled

#### Process Scanner Performance Issues
**Symptoms:**
- `scan_duration_ms` consistently high
- `scan_errors_total` increasing
- `limit_breaches_total` increasing

**Troubleshooting Steps:**
1. Check current process count via `process_count`
2. Review scan interval configuration
3. Check for high process churn rate
4. Verify CPU and memory usage of the scanner

**Resolution:**
- Increase scan interval to reduce frequency
- Add exclude patterns for unimportant processes
- Enable or adjust adaptive scanning parameters
- Increase `maxCPUUsage` threshold (with caution)
- Reduce `maxProcesses` limit to focus on important processes

#### Missing Process Data
**Symptoms:**
- Important processes not being tracked
- Gaps in process metrics
- Unexpected process_terminated events

**Troubleshooting Steps:**
1. Check include/exclude pattern configuration
2. Verify scanner is running (status is "running")
3. Look for notification errors in logs
4. Check event queue size for potential overflow

**Resolution:**
- Adjust include patterns to capture important processes
- Loosen exclude patterns that might be filtering too aggressively
- Increase event channel size if queue is filling up
- Increase event batch size for more efficient processing
- Ensure consumers are processing events efficiently

#### Low Capture Ratio
**Symptoms:**
- `topn_capture_ratio` metric consistently below 95%
- Missing important processes in telemetry data

**Troubleshooting Steps:**
1. Check `topn.maxProcesses` setting
2. Review process scoring weights
3. Look for high process churn rate

**Resolution:**
- Increase `topn.maxProcesses` to track more processes
- Adjust `topn.cpuWeight` and `topn.rssWeight` to better match environment
- Ensure `topn.churnHandlingEnabled` is true if operating in high-churn environment

#### Circuit Breaker Activation
**Symptoms:**
- `topn_circuit_breaker` metric value is 1
- `ModuleOverLimit` events in logs
- Reduced sampling or telemetry quality

**Troubleshooting Steps:**
1. Check `topn_update_time_seconds` and `sampler_cpu_percent`
2. Monitor `topn_churn_rate` for excessive process churn
3. Review recent system changes or load increases

**Resolution:**
- Increase `maxSamplerCPU` if system has available resources
- Adjust `topn.churnThreshold` to better match environment
- Increase `sampleInterval` to reduce update frequency
- Optimize process inclusion/exclusion rules

#### Statistical Accuracy Issues
**Symptoms:**
- Percentile calculations show unexpected values
- p95/p99 error rates exceed 1%
- Inconsistent aggregations across instances

**Troubleshooting Steps:**
1. Check `sketch_buckets` metric for potential overflow
2. Verify relative accuracy parameter (γ) is set correctly
3. Look for extreme values in the data (outliers)

**Resolution:**
- Adjust `ddSketch.relativeAccuracy` for better precision (lower values increase accuracy)
- Set appropriate `minValue` and `maxValue` to bound the range
- Increase `collapseThreshold` to reduce memory usage if needed
- Consider using dense store for more predictable memory usage

#### Memory Usage Growth
**Symptoms:**
- `sketch_memory_bytes` metric shows continuous growth
- Agent memory usage increasing over time

**Troubleshooting Steps:**
1. Monitor `sketch_buckets` metric
2. Check data distribution for high cardinality
3. Review store type (sparse vs. dense)

**Resolution:**
- Enable `autoSwitch` to automatically optimize for memory usage
- Decrease `collapseThreshold` to more aggressively collapse sparse buckets
- Adjust `switchThreshold` to switch to dense store more readily
- Consider using dense store directly for predictable memory usage

## Operational Scenarios
This section provides step-by-step guides for managing specific operational conditions.

### Configuring Process Scanner for High-Process Environments

In environments with a large number of processes (>1000), you may need to optimize the process scanner configuration:

1. **Assess current performance:**
   - Monitor `scan_duration_ms` to understand scan time
   - Check `process_count` to know how many processes are tracked
   - Monitor `cpu_usage_percent` to see resource impact

2. **Adjust scanning frequency:**
   - Increase `scanInterval` to reduce frequency (e.g., from 10s to 15s or 30s)
   - Enable `adaptiveSampling` to automatically adjust scan rate

3. **Filter unimportant processes:**
   - Add `excludePatterns` for short-lived or unimportant processes
   - Use `includePatterns` to focus only on specific processes of interest
   - Example patterns:
     - Exclude temporary processes: `"^.*\\_tmp.*$"`, `"^.*\\_temp.*$"`
     - Include only app processes: `"^app\\_.*$"`, `"^service\\_.*$"`

4. **Optimize resource usage:**
   - Set appropriate `maxProcesses` limit based on environment
   - Adjust `eventBatchSize` for efficient event processing
   - Increase `eventChannelSize` if many events occur rapidly

5. **Verify configuration:**
   - Monitor metrics after changes to ensure improvement
   - Check for any missed important processes
   - Validate that `scan_duration_ms` decreases
   - Confirm CPU usage stays below threshold

### Managing Process Scanner in Container Environments

Container environments have unique characteristics for process monitoring:

1. **Container-specific setup:**
   - Consider using container-specific include patterns to filter by cgroup
   - For Linux, ensure `procFSPath` points to the correct procfs location
   - Set `refreshCPUStats` to true for accurate container CPU accounting

2. **Handle high container churn:**
   - Enable `adaptiveSampling` for environments with frequent container creation/deletion
   - Set appropriate `maxScanTime` based on container density
   - Configure error handling with proper `retryInterval`

3. **Resource isolation:**
   - Set conservative `maxCPUUsage` to avoid impacting container workloads
   - Monitor `memory_usage_bytes` to ensure it stays within container limits
   - Use `limit_breaches_total` metric to detect when approaching resource limits

4. **Optimize for container orchestration:**
   - For Kubernetes environments, consider filtering by pod labels
   - In Docker environments, use naming patterns to focus on important containers
   - Handle container restart events properly with lifecycle monitoring

5. **Integration with container metrics:**
   - Correlate process metrics with container metrics
   - Map container IDs to process groups
   - Use container metadata to enrich process information

### ModuleOverLimit Resolution
When a module exceeds its resource limits, the agent will emit a `ModuleOverLimit` diagnostic event. Here's how to diagnose and resolve this situation:

**Scenario: Scanner or Sampler module exceeding CPU limit**

1. **Identify the issue:**
   - Check logs for `ModuleOverLimit` events specifying the module
   - Verify CPU usage metrics to confirm the breach

2. **Immediate mitigation:**
   - The agent will automatically activate the circuit breaker
   - The scanner will adapt its scanning rate
   - The sampler will reduce its sampling rate

3. **Root cause analysis:**
   - Check if there was a sudden increase in process creation rate
   - Verify if the host is under abnormal load
   - Check if collection frequency is set too high

4. **Resolution options:**
   - Adjust CPU thresholds in configuration
   - Modify scanning/sampling frequency
   - Update process inclusion/exclusion rules
   - Enable or tune adaptive sampling

5. **Verification:**
   - Monitor CPU usage after changes
   - Confirm absence of `ModuleOverLimit` events
   - Verify that telemetry quality remains within acceptable bounds

### High Process Churn Handling

In environments with high process churn (many short-lived processes), the Process Scanner and Top-N sampler may experience performance issues. Here's how to optimize for this scenario:

1. **Verify the issue:**
   - Check `topn_churn_rate` metric to confirm high churn rate
   - Look for frequent process creation/termination events
   - Monitor scan and update duration metrics

2. **Configure Process Scanner:**
   - Increase `scanInterval` to reduce impact of rapid changes
   - Use `excludePatterns` to filter short-lived processes
   - Enable `adaptiveSampling` for automatic adjustment
   - Increase `eventChannelSize` to handle event bursts

3. **Configure Top-N Sampler:**
   - Ensure `topn.churnHandlingEnabled` is set to `true`
   - Set `topn.churnThreshold` appropriate to your environment
   - Increase `topn.stabilityFactor` for more stable scoring
   - Adjust `topn.minScore` to ignore short-lived processes

4. **Verify improvements:**
   - Monitor CPU usage of both scanner and sampler
   - Check scan duration and update time metrics
   - Verify that important processes are still being captured
   - Confirm circuit breakers are not being triggered

5. **Long-term monitoring:**
   - Set up alerts for excessive churn rates
   - Monitor for changes in process patterns
   - Periodically review and refine exclusion patterns

### Optimizing DDSketch for Accuracy vs. Memory

When tuning the DDSketch for your specific environment, you may need to balance statistical accuracy against memory usage:

1. **Assess current performance:**
   - Check p95/p99 error rates against expected 1% threshold
   - Monitor `sketch_memory_bytes` metric for memory usage
   - Review `sketch_buckets` to understand bucket allocation

2. **For higher accuracy:**
   - Decrease `relativeAccuracy` parameter (e.g., from 0.0075 to 0.005)
   - Note: This will increase memory usage due to more buckets

3. **For lower memory usage:**
   - Increase `collapseThreshold` to merge more low-count buckets
   - Enable `autoSwitch` if not already enabled
   - Consider using dense store by setting `useSparseStore` to false
   - Tighten the range between `minValue` and `maxValue` if your data fits

4. **For highly skewed distributions:**
   - Start with sparse store (`useSparseStore` = true)
   - Set `autoSwitch` to true to adapt to the distribution
   - Adjust `switchThreshold` based on observed density metrics

5. **Verification and monitoring:**
   - Verify accuracy using test data with known percentiles
   - Monitor memory usage across all agent instances
   - Check for `StoreTypeSwitched` events in logs
   - Periodically review sketch configuration as data volume grows

### Distributed Aggregation Setup

When setting up distributed aggregation with DDSketch across multiple agents:

1. **Configure consistent parameters:**
   - Use identical `relativeAccuracy` parameter across all agents
   - Ensure `minValue` and `maxValue` are consistent

2. **Merging strategy:**
   - Collect serialized sketches from each agent
   - Deserialize and merge at aggregation point
   - Use `MergeBytes` for efficient network transport

3. **Transport optimization:**
   - Consider compression for serialized sketches
   - Use the sparse store for more efficient serialization
   - Monitor serialized size and adjust parameters if needed

4. **Verification:**
   - Compare aggregated results with raw data samples
   - Verify error bounds remain within expected limits
   - Check for any anomalies during merge operations

5. **Troubleshooting:**
   - If merge errors occur, verify parameter consistency
   - Check for extreme values that might be outside bounds
   - Verify all sketches use the same serialization version
