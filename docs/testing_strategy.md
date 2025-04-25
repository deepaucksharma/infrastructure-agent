# Infrastructure Agent Testing Strategy

## Overview

This document outlines the comprehensive testing strategy for the infrastructure agent components, with a particular focus on:

1. Process Scanner (collector/process_scanner.go)
2. OTLP Exporter (export/otlp/exporter.go)
3. Watchdog Module (watchdog/watchdog.go)

Our testing strategy emphasizes real-world validation through a 3-tier approach:

- **Component Tests (20%)**: Testing individual components with dependencies mocked
- **Integration Tests (10%)**: Testing components working together
- **System Tests (70%)**: Testing the complete system under realistic conditions

## Test Types and Locations

| Test Type | Location | Purpose |
|-----------|----------|---------|
| Component Tests | `collector/*_test.go`, `export/*_test.go`, `watchdog/*_test.go` | Validate individual component behavior with mocked dependencies |
| Integration Tests | `test/integration/*_test.go` | Validate components working together correctly |
| System Tests | `test/system/*_test.go` | Validate end-to-end functionality under realistic conditions |

## Running Tests

```bash
# Run unit tests (built alongside component code)
make test

# Run component tests 
make test-component

# Run integration tests
make test-integration

# Run system tests (short mode)
make test-system-short

# Run all tests
make test-all

# Run long-running stability test
make test-stability
```

## Test Tags

We use Go build tags to organize and selectively run tests:

```go
// Component test
//go:build component
// +build component

// Integration test
//go:build integration
// +build integration

// System test
//go:build system
// +build system
```

## Mocking Approach

We use testify/mock for creating mock implementations:

```go
// Example mock
type MockProcessCollector struct {
    mock.Mock
}

func (m *MockProcessCollector) GetProcesses() ([]*ProcessInfo, error) {
    args := m.Called()
    return args.Get(0).([]*ProcessInfo), args.Error(1)
}
```

## System Test Infrastructure

System tests rely on docker-compose to create a complete testing environment:

```bash
# Start test environment
./test/system/scripts/start_environment.sh

# Run system tests
go test -tags=system ./test/system/...

# Stop test environment
./test/system/scripts/stop_environment.sh
```

## G-Goal Test Coverage Matrix

| G-Goal | Test Types | Validation Methods |
|--------|------------|-------------------|
| G-1: Additive-Only Schema | Component, System | Schema validation with OTLP protocol |
| G-2: Host Safety | Component, System | CPU/memory monitoring during load tests |
| G-3: Statistical Fidelity | Component, Integration | Statistical accuracy validation |
| G-4: Top-N Accuracy | Integration, System | Capture ratio verification during system tests |
| G-5: Tail Error Bound | System | Statistical aggregation validation |
| G-6: Self-Governance | System | Failure injection, recovery testing |
| G-7: Documentation Parity | System | Automated validation of documented features |
| G-8: Local Tests Green | All | CI verification with all test types |
| G-9: Architectural Conformance | Component, Integration | Interface compliance verification |
| G-10: Documentation Quality | System | Runbook scenario execution |

## Test Data Management

Tests use the following data strategies:

1. **Synthetic Test Data**: Generated programmatically for predictable test scenarios
2. **Real-World Samples**: Anonymized snapshots of production data for realistic tests
3. **Docker-Based Environments**: Containers for simulating complex environments

## Continuous Testing

Tests are run at different stages:

1. **Pre-Commit**: Unit/Component tests 
2. **CI Pipeline**: All test types except long-running stability tests
3. **Nightly Builds**: Complete test suite including system tests
4. **Weekly**: Long-running stability tests

## Test Metrics and Success Criteria

| Metric | Target | Validation Method |
|--------|--------|-------------------|
| Process Scanner CPU | ≤0.75% | System tests with real processes |
| Process Scanner Memory | ≤30MB | System tests under memory pressure |
| OTLP Exporter Throughput | ≥10,000 events/s | Performance benchmarks |
| End-to-End Latency | ≤10ms per event | System tests with timing checks |
| Test Coverage | ≥85% | Coverage reports in CI |