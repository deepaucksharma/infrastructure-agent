# Infrastructure Agent Testing

This directory contains comprehensive tests for the infrastructure agent, with a focus on the Process Scanner, OTLP Exporter, and Watchdog components.

## Test Types

Our testing approach is organized into three main categories:

1. **Component Tests (20%)** - Test individual components with mocked dependencies
2. **Integration Tests (10%)** - Test components working together
3. **System Tests (70%)** - Test the complete system under realistic conditions

## Directory Structure

```
infrastructure-agent/
├── collector/                   # Process Scanner component
│   └── *_test.go               # Component tests for Process Scanner
├── export/                      # OTLP Exporter component
│   └── *_test.go               # Component tests for OTLP Exporter
├── test/
│   ├── integration/            # Integration tests
│   │   └── *_test.go           # Tests for component interactions
│   ├── system/                 # System tests
│   │   ├── *_test.go           # Full system tests
│   │   ├── config/             # Test environment configuration
│   │   ├── docker-compose.yml  # Test environment definition
│   │   └── scripts/            # Test environment management scripts
│   └── Makefile.testing        # Test execution targets
└── watchdog/                   # Watchdog component
    └── *_test.go               # Component tests for Watchdog
```

## Running Tests

### Prerequisites

- Go 1.19 or later
- Docker and Docker Compose (for system tests)
- Make

### Component Tests

Component tests validate individual components with dependencies mocked.

```bash
make test-component
```

### Integration Tests

Integration tests validate components working together correctly.

```bash
make test-integration
```

### System Tests

System tests validate the complete system under realistic conditions using Docker containers to simulate a production environment.

```bash
# Start the test environment
cd test/system/scripts
./start_environment.sh

# Run system tests (short mode)
make test-system-short

# Run full system tests (longer)
make test-system-full

# Stop the test environment
cd test/system/scripts
./stop_environment.sh
```

### Complete Test Suite

Run all tests (except long stability tests):

```bash
make test-all
```

### G-Goal Validation

Specific tests for validating blueprint goals:

```bash
make test-g-goals
```

## Testing Strategy

Our testing strategy emphasizes real-world validation through system tests (70% of coverage) while maintaining adequate component (20%) and integration (10%) testing for faster feedback cycles.

The tests verify:

1. **Functional Correctness**: All components perform their intended functions
2. **Performance Efficiency**: Components meet performance targets (CPU ≤0.75%, memory optimized)
3. **Reliability**: Components function under varied conditions (high process counts, system stress, errors)
4. **Blueprint Compliance**: Implementation satisfies all G-Goals

## G-Goal Test Coverage

| G-Goal | Test Types | Test Files |
|--------|------------|------------|
| G-1: Additive-Only Schema | Component, System | `export/otlp_exporter_test.go`, `test/system/otlp_schema_test.go` |
| G-2: Host Safety | Component, System | `collector/process_scanner_test.go`, `test/system/process_scanner_system_test.go` |
| G-3: Statistical Fidelity | Component, Integration | `sketch/ddsketch_test.go`, `test/integration/scanner_topn_integration_test.go` |
| G-4: Top-N Accuracy | Integration, System | `test/integration/scanner_topn_integration_test.go`, `test/system/topn_system_test.go` |
| G-5: Tail Error Bound | System | `test/system/statistical_validation_test.go` |
| G-6: Self-Governance | System | `test/system/process_scanner_system_test.go`, `test/system/watchdog_system_test.go` |
| G-7: Documentation Parity | System | `test/system/documentation_parity_test.go` |
| G-8: Local Tests Green | All | All test files |
| G-9: Architectural Conformance | Component, Integration | Various component and integration tests |
| G-10: Documentation Quality | System | `test/system/runbook_validation_test.go` |

## Contributing

When adding new tests:

1. Use the appropriate build tags:
   - `//go:build component` for component tests
   - `//go:build integration` for integration tests
   - `//go:build system` for system tests

2. Follow the existing patterns and naming conventions

3. Ensure tests validate specific G-Goals where applicable

4. Add any required test fixtures or mock implementations