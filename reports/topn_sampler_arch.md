# Architecture Review: topn_sampler

## Blueprint Alignment

### Epic/Story Mapping
- **Epic**: A1-E1 - Process Collection & Scoring
- **Story**: A1-S3 - Adaptive-N Top-Heap

### Module Interface Conformance
- [x] Implements `Resources()` interface - Will return resource usage metrics for the sampler
- [x] Implements `Shutdown()` interface - Will gracefully stop sampling
- [x] Implements `Init()` interface - Will initialize the sampler with configuration
- [x] Conforms to error handling patterns - Will use standard error patterns

### Data Flow Analysis
- **Inputs**: 
  - Process information from system (PID, CPU, RSS, cmd)
  - Configuration parameters (sample rate, N, weights)
- **Outputs**: 
  - Top-N processes with scores
  - Metrics on capture ratio and performance
  - Diagnostic events for circuit breaker activation
- **Dependencies**: 
  - Core agent configuration
  - Process information collector
  - Metrics emitter

## Implementation Approach

### Algorithms & Data Structures
- **Min-Heap**: O(log N) operations for maintaining Top-N processes
- **Scoring Function**: Weighted combination of CPU and RSS with configurable weights
- **Circular Buffer**: For calculating rate-of-change metrics (O(1) operations)
- **Circuit Breaker Pattern**: For self-protection under high load

### Resource Budgeting
- **CPU Budget**: 0.5% of total agent allocation (G-2 specifies 2% total)
- **Memory Budget**: 10MB maximum RSS (G-2 specifies 100MB total)
- **I/O Considerations**: Minimal disk I/O, only for metrics emission

### Critical Sections
- **Heap Operations**: Protected by read-write mutex for concurrent access
- **Configuration Updates**: Atomic operations for thread safety

## Potential Concerns

### Risk Areas
- **Process Churn Handling**: High churn rate (2000 PIDs/s) could impact performance
  - Mitigation: Optimized hash table for PID lookups, batch update operations
- **Memory Management**: Tracking too many processes could exceed memory budget
  - Mitigation: Fixed-size data structures, automatic pruning of low-scoring processes
- **CPU Spikes**: Scoring calculation could cause CPU spikes
  - Mitigation: Incremental updates, circuit breaker pattern

### Blueprint Deviations
- No deviations from blueprint required at this time

## Architecture Decision Records

### ADR References
- ADR-001: The statistical fidelity requirements (γ = 0.0075) will be respected in the implementation

## Final Assessment
- [x] Implementation approach fully aligns with blueprint
- [x] Implementation meets all acceptance criteria
- [x] Implementation respects all G-goals
- [x] Implementation contains no unnecessary complexity

### Approval
Implementation approach is: APPROVED

### Notes for Implementation
1. Ensure proper instrumentation for detecting and reporting CPU usage
2. Implement circuit breaker with configurable thresholds
3. Consider using an LRU cache for recently exited processes to handle rapid process cycling
4. Add explicit metrics for capture ratio to verify 95% requirement
5. Ensure all operations have predictable and bounded complexity
