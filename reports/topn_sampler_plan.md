# Work Plan: topn_sampler

## Overview
Implement an efficient Top-N process sampler that maintains high capture ratio (≥95%) while keeping CPU usage low (≤0.5%). The sampler will use a heap-based approach to efficiently track the most resource-intensive processes on the system.

## Blueprint Alignment
- **Epic**: A1-E1 - Process Collection & Scoring
- **Story**: A1-S3 - Adaptive-N Top-Heap
- **Acceptance Criteria**:
  - topn_capture_ratio ≥ 95% on replay harness
  - sampler CPU ≤ 0.5%
  - heap operations O(k log N) verified by profiling
  - capture ratio stable under PID churn (2000 PIDs/s)

## Implementation Plan

### 1. Files to Modify
- `infrastructure-agent/sampler/sampler.go`: Add interface definition for samplers
- `infrastructure-agent/sampler/registry.go`: Registry for managing samplers
- `infrastructure-agent/sampler/config.go`: Configuration structure and defaults

### 2. New Files to Create
- `infrastructure-agent/sampler/topn.go`: Main implementation of Top-N sampler
- `infrastructure-agent/sampler/topn_test.go`: Unit tests for Top-N sampler
- `infrastructure-agent/sampler/heap.go`: Min-heap implementation for process tracking
- `infrastructure-agent/sampler/heap_test.go`: Unit tests for heap implementation
- `infrastructure-agent/sampler/process.go`: Process metadata structure
- `infrastructure-agent/sampler/metrics.go`: Metrics for sampler performance

### 3. Implementation Steps
1. Define the interfaces and core data structures for the sampler module
2. Implement the process metadata structure with scoring function
3. Create an efficient min-heap implementation for process tracking
4. Implement the Top-N sampler with configurable parameters:
   - Maximum processes to track (N)
   - Scoring function (weighted CPU/RSS combination)
   - Sampling interval
   - Resource limits
5. Add circuit breaker to reduce sampling under high load
6. Implement metrics for tracking capture ratio and resource usage
7. Add configuration parameters with sensible defaults
8. Write comprehensive unit tests for all components
9. Create integration tests for end-to-end validation
10. Update documentation and changelog

### 4. Test Plan
- **Unit Tests**:
  - Test heap operations (insert, remove, update) - verify O(log N)
  - Test process scoring with various CPU/RSS values
  - Test Top-N sampler with static process list
  - Test sampler with process churn (add/remove)
  - Test circuit breaker activation under high load
  - Test configuration validation

- **Integration Tests**:
  - Run with replay corpus to verify capture ratio
  - Test with high PID churn scenario (2000 PIDs/s)
  - Measure CPU usage under various loads
  - Test behavior with different configurations

- **Performance Tests**:
  - Benchmark heap operations with various sizes
  - Benchmark process scoring function
  - Benchmark sampler update with different N values
  - Profile CPU usage during high churn scenarios

### 5. Documentation Updates
- `docs/runbook.md`: Add section on Top-N sampler, configuration parameters, and troubleshooting
- `docs/CHANGELOG.md`: Add entry for Top-N sampler implementation

## Potential Risks & Mitigations
- **Risk**: High process churn causing excessive CPU usage
  - **Mitigation**: Implement adaptive sampling rate based on system load
- **Risk**: Memory leaks from process tracking
  - **Mitigation**: Ensure proper cleanup of exited processes and use weak references
- **Risk**: Capture ratio drops during high activity
  - **Mitigation**: Use predictive scoring to anticipate important processes

## Success Metrics
- topn_capture_ratio: ≥95% in harness tests
- CPU Usage: ≤0.5% during normal operation
- Memory Usage: ≤10MB for sampler module
- Stability: No degradation under 2000 PIDs/s churn

## Estimated Effort
- Implementation: 2-3 days
- Testing: 1-2 days
- Documentation: 0.5 days
