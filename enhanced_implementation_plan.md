# Enhanced AI-Infrastructure Agent Implementation Plan

*(Incorporating Comprehensive E2E Testing Framework)*

---

## 0. Objective & Non-Negotiable Goals

From an **empty folder** on a fresh machine, an LLM-powered engineer ("**InfraBuilder**") must:
1. Provision a minimal development environment.
2. Clone only the necessary source code (**infrastructure-agent**) and its **testbed**.
3. Read the blueprint & ADR to internalize **purpose**, **constraints**, and **success metrics**.
4. Iteratively implement, test, and document changes – **entirely offline**.
5. Maintain strict alignment with the blueprint architecture at every step.

### 0.1. Hard-Stop Goals & Guard-Rails *(InfraBuilder must never violate these)*

| #       | Goal                             | Concrete Check                                            | Verification Method                                     |
|---------|----------------------------------|-----------------------------------------------------------|--------------------------------------------------------|
| **G-1** | **Additive-Only Schema**         | `.proto` diffs pass `buf breaking` with zero changes      | `buf breaking --against "HEAD^"` after schema changes   |
| **G-2** | **Host Safety**                  | CPU ≤ 2% and RSS ≤ 100 MB on 500-PID harness run          | Resource monitor in harness with explicit thresholds    |
| **G-3** | **Statistical Fidelity**         | DDSketch p95/p99 error ≤ 1% (γ = 0.0075)                  | Error calculation across replay corpus                  |
| **G-4** | **Top-N Accuracy**               | `topn_capture_ratio` ≥ 95% in harness output              | Explicit count verification in output logs              |
| **G-5** | **Tail Error Bound**             | Aggregated CPU/RSS sums differ ≤ 5% from baseline         | Direct comparison with baseline metrics                 |
| **G-6** | **Self-Governance**              | Breakers/fallbacks emit `AgentDiagEvent`                  | Log scanning for expected events during failure tests   |
| **G-7** | **Documentation Parity**         | All new features documented in `docs/runbook.md`          | Feature-to-docs mapping verification                    |
| **G-8** | **Local Test Green**             | All test commands exit with code 0                        | Automated test suite execution                          |
| **G-9** | **Architectural Conformance**    | Implementations follow blueprint patterns                 | Architecture review before and after implementation     |
| **G-10**| **Documentation Quality**        | Runbook validates operational scenarios                   | Scenario-driven runbook validation                      |

If **any** goal fails, InfraBuilder must iterate until all checks pass before moving a task to `tasks/done/`.

---

## 1. Environment Setup

### 1.1. System Tools (install in PATH)

| Tool | Version | Reason |
|------|---------|--------|
| Git CLI | Latest | Clone/update repositories |
| Go | ≥ 1.22 | Build infrastructure-agent & tests |
| Docker & Compose | Latest | Spin up Kafka, Redis, NRDB stub in testbed |
| Protocol Buffers (`protoc`) | Latest | Compile protobuf schemas |
| `buf` | Latest | Validate protobuf schema compatibility |
| GNU Make | Latest | Convenience wrappers (lint, test, harness) |
| Kind | Latest | Kubernetes-in-Docker for E2E testing |
| Firecracker | Latest | Microserver VM for Linux testing |
| ToxiProxy | Latest | Network failure simulation |

> *Optional (only when merger later moves local)*: Java 17 + Maven/Gradle.

### 1.2. Folder Layout – after bootstrap

```
workspace/
  infrastructure-agent/            # main Go codebase
  infrastructure-agent-testbed/    # docker-compose harness & replay corpus
  docs/                            # blueprint, ADR, run-books
    blueprint_vFinal.md            # architectural blueprint
    ADR-001.md                     # additive schema decision
    runbook.md                     # operational documentation
    CHANGELOG.md                   # version change log
  tasks/                           # task definitions (one per feature/bug)
    pending/                       # tasks to be implemented
    in_progress/                   # tasks currently being worked on
    done/                          # completed and verified tasks
  reports/                         # detailed implementation logs
    <slug>_plan.md                 # planning document for each task
    <slug>_arch.md                 # architecture review notes
    <slug>_result.md               # implementation results and metrics
  e2e/                             # enhanced end-to-end testing framework
    scenarios/                     # test scenario definitions
    kind/                          # kind cluster configuration
    workload/                      # workload generation system
    chaos/                         # failure injection framework
    validators/                    # blueprint goal validators
    profiling/                     # resource profiling system 
    reporting/                     # result reporting system
  blueprint_tasks.yaml             # mapping between blueprint and tasks
  Makefile                         # local testing shortcuts
  checkpoint.sh                    # verification script for goals
```

---

## 2. Bootstrap Process

### 2.1. Enhanced Bootstrap Script (`bootstrap.sh`)

```bash
#!/usr/bin/env bash
set -e
mkdir -p workspace && cd workspace

# 1. Clone code & testbed
[ ! -d infrastructure-agent ] && \
  git clone https://github.com/newrelic/infrastructure-agent.git
[ ! -d infrastructure-agent-testbed ] && \
  git clone https://github.com/newrelic/infrastructure-agent-testbed.git

# 2. Set up documentation structure
mkdir -p docs
cp ../Blueprint_vFinal.md docs/blueprint_vFinal.md
cp ../ADR-001.md docs/ADR-001.md
touch docs/runbook.md
touch docs/CHANGELOG.md

# 3. Create task and report directories
mkdir -p tasks/pending tasks/in_progress tasks/done reports

# 4. Set up testbed containers
cd infrastructure-agent-testbed && docker compose pull && docker compose up -d
cd ..

# 5. Create E2E testing framework structure
mkdir -p e2e/scenarios e2e/kind e2e/workload e2e/chaos e2e/validators e2e/profiling e2e/reporting

# 6. Create Blueprint-Task mapping
cat > blueprint_tasks.yaml <<'EOF'
# Blueprint to Task mapping
epics:
  A1-E1:
    title: "Process Collection & Scoring"
    stories:
      A1-S3:
        title: "Adaptive-N Top-Heap"
        tasks:
          - slug: topn_sampler
            phase: P0
            module: sampler/
EOF

# 7. Create enhanced Makefile with E2E test commands
cat > Makefile <<'EOF'
lint:
	cd infrastructure-agent && golangci-lint run ./...

test:
	cd infrastructure-agent && go test ./... -race -cover

bench:
	cd infrastructure-agent && go test -run=^$ -bench=. ./sampler/... ./sketch/... > ../_bench.txt

harness:
	cd infrastructure-agent-testbed && ./run_replay.sh

integration:
	cd infrastructure-agent-testbed && ./run_integration.sh

e2e:
	cd e2e && ./e2e_run.sh --scenario=standard

e2e-matrix:
	cd e2e && ./e2e_run.sh --matrix

chaos:
	cd e2e && ./e2e_run.sh --scenario=failover

verify:
	./checkpoint.sh
EOF

# 8. Create enhanced checkpoint verification script with more comprehensive checks
cat > checkpoint.sh <<'EOF'
#!/bin/bash
# Goal verification script
echo "Verifying implementation against goals G-1 through G-10..."

# G-1: Additive-Only Schema
if [ -f infrastructure-agent/proto/*.proto ]; then
  cd infrastructure-agent && buf breaking --against "HEAD^" && cd ..
  if [ $? -ne 0 ]; then
    echo "❌ G-1 FAIL: Breaking proto changes detected"
    exit 1
  else
    echo "✅ G-1 PASS: No breaking proto changes"
  fi
fi

# G-2: Host Safety
cd e2e && ./validate_goal.sh G-2
if [ $? -ne 0 ]; then
  echo "❌ G-2 FAIL: Host safety requirements not met"
  exit 1
else
  echo "✅ G-2 PASS: Host safety requirements met"
fi

# Continue with enhanced checks for other goals...
EOF
chmod +x checkpoint.sh

# 9. Initialize kind configuration for E2E testing
cat > e2e/kind/cluster.yaml <<'EOF'
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraMounts:
  - hostPath: ../../infrastructure-agent
    containerPath: /infrastructure-agent
EOF

# 10. Create sample E2E test scenario
cat > e2e/scenarios/standard.yml <<'EOF'
name: standard
blueprint_goals:
  - G-1  # Additive-Only Schema
  - G-2  # Host Safety
  - G-3  # Statistical Fidelity
  - G-4  # Top-N Accuracy
  - G-5  # Tail Error Bound
  - G-8  # Local Test Green
  - G-9  # Architectural Conformance

platforms:
  - linux:
      kernels: [5.10]
      
workload:
  type: standard
  params:
    pids: 500
    duration_minutes: 10
    
configuration:
  base: standard
    
agents:
  count: 1
  
validation:
  kpis:
    topn_capture_ratio:
      min: 95.0
      unit: percent
    p95p99_error:
      max: 1.0
      unit: percent
    cpu_usage:
      max: 2.0
      unit: percent
    memory_usage:
      max: 100
      unit: MB
EOF

# 11. Create E2E run script
cat > e2e/e2e_run.sh <<'EOF'
#!/bin/bash
# E2E Test Runner

SCENARIO="standard"
MATRIX=false

# Parse arguments
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    --scenario=*)
    SCENARIO="${key#*=}"
    shift
    ;;
    --matrix)
    MATRIX=true
    shift
    ;;
    *)
    echo "Unknown option: $key"
    exit 1
    ;;
  esac
done

# Set paths
SCENARIO_PATH="scenarios/${SCENARIO}.yml"
RESULTS_DIR="results/$SCENARIO"

# Create results directory
mkdir -p "$RESULTS_DIR"

echo "Running E2E test scenario: $SCENARIO"

# 1. Environment Setup
echo "Setting up test environment..."
# Setup code here

# 2. Test Execution
echo "Executing tests..."
# Test execution code here

# 3. Result Collection
echo "Collecting results..."
# Result collection code here

# 4. Validation
echo "Validating results..."
# Validation code here
./validate_goal.sh G-2
./validate_goal.sh G-3
./validate_goal.sh G-4
./validate_goal.sh G-5

# 5. Reporting
echo "Generating report..."
# Reporting code here

echo "E2E tests complete. See results in: $RESULTS_DIR"
EOF
chmod +x e2e/e2e_run.sh

# 12. Create goal validator script
cat > e2e/validate_goal.sh <<'EOF'
#!/bin/bash
# Blueprint Goal Validator

GOAL="$1"

case "$GOAL" in
  "G-1")
    # Additive-Only Schema
    cd ../infrastructure-agent && buf breaking --against "HEAD^"
    exit $?
    ;;
  "G-2")
    # Host Safety
    # Check CPU and memory usage
    cpu_ok=$(grep "CPU Usage" "$RESULTS_DIR/metrics.txt" | awk '{print ($3 <= 2.0)}')
    mem_ok=$(grep "Memory Usage" "$RESULTS_DIR/metrics.txt" | awk '{print ($3 <= 100.0)}')
    
    if [[ "$cpu_ok" == "1" && "$mem_ok" == "1" ]]; then
      exit 0
    else
      exit 1
    fi
    ;;
  "G-3")
    # Statistical Fidelity
    error=$(grep "p95/p99 Error" "$RESULTS_DIR/metrics.txt" | awk '{print $3}')
    if (( $(echo "$error <= 1.0" | bc -l) )); then
      exit 0
    else
      exit 1
    fi
    ;;
  "G-4")
    # Top-N Accuracy
    ratio=$(grep "topn_capture_ratio" "$RESULTS_DIR/metrics.txt" | awk '{print $2}')
    if (( $(echo "$ratio >= 95.0" | bc -l) )); then
      exit 0
    else
      exit 1
    fi
    ;;
  "G-5")
    # Tail Error Bound
    error=$(grep "Tail Error" "$RESULTS_DIR/metrics.txt" | awk '{print $3}')
    if (( $(echo "$error <= 5.0" | bc -l) )); then
      exit 0
    else
      exit 1
    fi
    ;;
  *)
    echo "Unknown goal: $GOAL"
    exit 1
    ;;
esac
EOF
chmod +x e2e/validate_goal.sh

echo "Enhanced bootstrap complete. Ready to begin implementation with E2E testing capabilities."
```

---

## 3. Task Management System

### 3.1. Enhanced Task File Format

```yaml
# tasks/pending/topn_sampler.yaml
slug: topn_sampler
phase: P0
module: sampler/
blueprint:
  epic: A1-E1
  story: A1-S3
acceptance:
  - topn_capture_ratio ≥ 95% on replay harness
  - sampler CPU ≤ 0.5%
  - heap operations O(k log N) verified by profiling
  - capture ratio stable under PID churn (2000 PIDs/s)
e2e_scenarios:
  - standard
  - high_pid_churn
blueprint_goals:
  - G-2  # Host Safety
  - G-3  # Statistical Fidelity
  - G-4  # Top-N Accuracy
```

### 3.2. Blueprint-Task Alignment

The `blueprint_tasks.yaml` file ensures each task is explicitly linked to the architectural blueprint and associated E2E test scenarios. For each new task:

1. Verify the task exists in the blueprint
2. Map the task to specific epic/story components
3. Identify which E2E scenarios should validate this task
4. List which blueprint goals are validated by this task
5. Ensure implementation properly addresses all blueprint requirements

---

## 4. Implementation Workflow

### 4.1. Enhanced Task Loop

For each task file in `tasks/pending/`:

1. **OPEN**: Move task file to `tasks/in_progress/` and parse `slug`, `phase`, `module`, acceptance criteria, and associated E2E scenarios and blueprint goals.

2. **PLAN**: Create a detailed work plan in `reports/<slug>_plan.md`:
   - Map implementation to blueprint components
   - List specific files to modify
   - Outline algorithms and data structures
   - Identify potential risks and mitigation strategies
   - Define test cases covering all acceptance criteria
   - Specify which E2E scenarios will validate this implementation

3. **ARCHITECTURE REVIEW**: Before coding, validate approach against blueprint:
   - Document in `reports/<slug>_arch.md`
   - Verify module interfaces align with blueprint
   - Confirm implementation follows code patterns
   - Check for potential side effects on other components
   - Validate resource budget allocation
   - Ensure E2E tests will validate all relevant blueprint goals

4. **CODE**: Implement changes while maintaining blueprint alignment:
   - Modify `infrastructure-agent` source
   - Add/update unit and integration tests
   - Enhance or create E2E test scenarios as needed
   - Update `docs/runbook.md` with new features
   - Add entries to `docs/CHANGELOG.md`

5. **TEST**: Execute comprehensive test suite:
   - `make lint` for static analysis
   - `make test` for unit tests (must achieve ≥80% coverage)
   - `make bench` for performance benchmarks
   - `make harness` for end-to-end validation, including:
     - Standard replay corpus
     - High-PID Churn scenario (≥2000 PID events/s)
     - eBPF Fallback test on specified kernel versions
     - Network Latency simulation for OTLP export
   - `make e2e` for enhanced E2E validation across platforms:
     - Blueprint goal validation
     - Multi-platform compatibility testing
     - Failure injection scenarios
     - Long-running stability tests

6. **INTEGRATION**: Validate end-to-end functionality:
   - Deploy against downstream components
   - Run integration scenarios:
     - `--scenario=dualwrite` (serialization validation)
     - `--scenario=nrdb_write` (storage validation)
     - `--scenario=query_router_fallback` (fallback validation)
   - Verify metrics and events are correctly transmitted

7. **GOAL VERIFICATION**: Run enhanced `checkpoint.sh` to validate all goals:
   - Verify each goal G-1 through G-10 independently using E2E framework
   - Log detailed metrics for each criterion
   - Flag any deviation from expected thresholds
   - Generate comprehensive validation report

8. **DOCUMENT**: Create detailed results in `reports/<slug>_result.md`:
   - Report pass/fail status for each test
   - Document performance metrics and deltas
   - Include diagnostic test results
   - Record any architectural insights or observations
   - Validate runbook through specific operational scenarios
   - Include relevant E2E test results and goal validations

9. **ITERATE**: If any goal fails or architecture misaligns:
   - Identify root causes of failures
   - Revise implementation approach
   - Update tests as needed
   - Re-run verification until all checks pass

10. **COMPLETE**: When all criteria are met:
    - Move task file to `tasks/done/`
    - Update `blueprint_tasks.yaml` with completion status
    - Generate progress report against blueprint
    - Update E2E test coverage matrix
    - Proceed to next task

Continue this loop until no tasks remain in `tasks/pending/`.

---

## 5. Enhanced Testing & Validation Framework

### 5.1. Multi-Level Test Matrix

| Level | Command | Purpose | Specific Tests | Pass Criteria |
|-------|---------|---------|----------------|---------------|
| **Unit** | `make lint` | Static code analysis | - Go linting<br>- Proto linting<br>- Code style checks | No errors or warnings |
| **Component** | `make test` | Unit & integration tests | - Core functionality<br>- Edge cases<br>- Race conditions<br>- Error handling | 100% pass, ≥80% coverage |
| **Performance** | `make bench` | Performance benchmarking | - Heap operations<br>- Sketch operations<br>- Serialization/deserialization<br>- Memory allocation patterns | Within time/space budgets |
| **System** | `make harness` | End-to-end simulation | - Standard replay corpus<br>- High-PID scenario<br>- eBPF fallback<br>- Network latency<br>- Key explosion | All metrics within thresholds |
| **Multi-Platform** | `make e2e` | Multi-platform validation | - Linux variants<br>- Windows variants<br>- macOS validation | All platforms passing |
| **Chaos** | `make chaos` | Resilience testing | - Network failures<br>- Service outages<br>- Resource constraints | Proper fallback behavior |
| **Compliance** | `make verify` | Goal verification | - G-1 through G-10 checks | All goals verified |

### 5.2. Multi-Dimensional E2E Scenarios

Enhanced E2E testing across multiple dimensions:

1. **Platforms**:
   - Linux with different kernel versions (4.14 → 6.8)
   - Windows Server versions (2016 → 2022)
   - macOS when available

2. **Workloads**:
   - **standard**: Baseline workload with mixed processes
   - **high_pid_churn**: Rapid process creation/deletion (2000 PIDs/s)
   - **memory_pressure**: System under memory constraints
   - **cpu_spikes**: Periodic CPU utilization spikes
   - **high_cardinality**: Metrics with many dimensions

3. **Configurations**:
   - **minimal**: Essential settings only
   - **standard**: Default production configuration
   - **full**: All features enabled
   - **custom**: Task-specific configuration

4. **Failure Modes**:
   - **network_latency**: Increased OTLP network latency
   - **service_restart**: Kafka/Redis/OTLP collector restarts
   - **resource_constraints**: CPU/memory limits
   - **data_corruption**: Invalid data handling

### 5.3. Automated Goal Validation

Explicit mapping of tests to blueprint goals:

```go
// e2e/validators/goal_validators.go
package validators

// GoalValidator defines the interface for validating blueprint goals
type GoalValidator interface {
    // Name returns the goal identifier (e.g., "G-2")
    Name() string
    
    // Description returns a human-readable description
    Description() string
    
    // Validate checks if the goal is met
    Validate(ctx context.Context) (bool, string, error)
}

// HostSafetyValidator validates G-2 (Host Safety)
type HostSafetyValidator struct {
    cpuThreshold    float64
    memoryThreshold float64
    resourceMonitor *ResourceMonitor
}

func (v *HostSafetyValidator) Name() string {
    return "G-2"
}

func (v *HostSafetyValidator) Description() string {
    return "Host Safety - CPU ≤ 2% and RSS ≤ 100 MB on 500-PID harness run"
}

func (v *HostSafetyValidator) Validate(ctx context.Context) (bool, string, error) {
    stats, err := v.resourceMonitor.GetAggregateStats(ctx)
    if err != nil {
        return false, "", err
    }
    
    cpuValid := stats.CPUPercent <= v.cpuThreshold
    memValid := stats.MemoryMB <= v.memoryThreshold
    
    if cpuValid && memValid {
        return true, fmt.Sprintf("CPU: %.2f%%, Memory: %.2fMB", stats.CPUPercent, stats.MemoryMB), nil
    }
    
    return false, fmt.Sprintf("Goal G-2 failed: CPU: %.2f%% (max %.2f%%), Memory: %.2fMB (max %.2fMB)",
        stats.CPUPercent, v.cpuThreshold, stats.MemoryMB, v.memoryThreshold), nil
}
```

### 5.4. Enhanced Resource Monitoring

Advanced CPU/memory monitoring across all test environments:

```go
// e2e/profiling/resource_profiler.go
package profiling

// ResourceProfiler collects and analyzes resource usage data
type ResourceProfiler struct {
    // ...
}

// StartProfiling begins resource profiling for a process
func (p *ResourceProfiler) StartProfiling(pid int, options ProfileOptions) error {
    // Attach profiler to target process
    // For Go processes, use pprof HTTP endpoints
    // For other processes, use OS-specific profiling tools
    
    // Set up periodic sampling
    // ...
}

// AnalyzeProfile analyzes collected profile data
func (p *ResourceProfiler) AnalyzeProfile(profile ProfileData) ProfileAnalysis {
    // Calculate resource usage statistics
    // Identify hotspots and anomalies
    // Compare against baseline and thresholds
    // ...
    
    return analysis
}

// GenerateReport creates a detailed profile report
func (p *ResourceProfiler) GenerateReport(analysis ProfileAnalysis) (*ProfileReport, error) {
    // Create report with charts and annotations
    // Highlight any areas of concern
    // Include recommendations
    // ...
    
    return report, nil
}
```

### 5.5. Chaos Engineering Framework

Comprehensive failure injection system:

```go
// e2e/chaos/controller.go
package chaos

// FailureType represents the type of failure to inject
type FailureType string

const (
    NetworkLatency      FailureType = "network_latency"
    NetworkPartition    FailureType = "network_partition"
    ServiceRestart      FailureType = "service_restart"
    ResourceConstraint  FailureType = "resource_constraint"
    DataCorruption      FailureType = "data_corruption"
    ClockSkew           FailureType = "clock_skew"
)

// Controller manages failure injections
type Controller struct {
    // ...
}

// InjectFailure injects a failure into the system
func (c *Controller) InjectFailure(failureType FailureType, target string, params map[string]interface{}, duration time.Duration) (string, error) {
    // Create failure injection
    // Apply to target component
    // Schedule resolution (if auto-resolve)
    // ...
    
    return failureID, nil
}

// ResolveFailure resolves an injected failure
func (c *Controller) ResolveFailure(failureID string) error {
    // Remove failure condition
    // Verify system stability
    // ...
    
    return nil
}
```

---

## 6. Enhanced Documentation & Monitoring System

### 6.1. Comprehensive Runbook Structure

Every feature implementation must update the runbook with:

1. **Feature Overview**:
   - Purpose and function
   - Configuration parameters
   - Default settings and valid ranges

2. **Metrics and Events**:
   - New metrics emitted
   - Event types and trigger conditions
   - Normal ranges and interpretation

3. **Diagnostic Procedures**:
   - Symptoms of common issues
   - Troubleshooting steps
   - Remediation actions

4. **Operational Scenarios**:
   - Step-by-step guide for managing specific conditions
   - Example: Diagnosing and resolving `ModuleOverLimit`

5. **Blueprint Goal Coverage**:
   - Which blueprint goals this feature addresses
   - How to verify goal compliance
   - Key metrics for goal validation

### 6.2. Enhanced Scenario-Based Validation

Each runbook entry is validated through multiple scenarios:

```yaml
# e2e/scenarios/runbook_scenarios.yml
name: runbook_scenarios
blueprint_goals:
  - G-6  # Self-Governance
  - G-7  # Documentation Parity
  - G-10 # Documentation Quality
  
platforms:
  - linux:
      kernels: [5.10]
      
scenarios:
  - name: module_overlimit
    description: "Test agent behavior when a module exceeds resource limits"
    steps:
      - action: configure
        params:
          maxSamplerCPU: 0.1
      - action: start_workload
        params:
          type: high_pid_churn
          pids_per_second: 2000
      - action: wait
        params:
          duration_seconds: 60
      - action: verify
        params:
          expect_event: "AgentDiagEvent.*ModuleOverLimit.*sampler"
      - action: configure
        params:
          maxSamplerCPU: 0.5
      - action: wait
        params:
          duration_seconds: 60
      - action: verify
        params:
          expect_no_event: "AgentDiagEvent.*ModuleOverLimit"
          
  - name: sketch_fallback
    description: "Test fallback to raw data when sketch encounters issues"
    steps:
      - action: configure
        params:
          sketch_enable: true
      - action: inject_failure
        params:
          type: service_restart
          target: sketch-merger
      - action: wait
        params:
          duration_seconds: 30
      - action: verify
        params:
          expect_event: "AgentDiagEvent.*SketchFallback"
          expect_metric: "raw_writes"
          
validation:
  runbook_coverage:
    required_sections:
      - "ModuleOverLimit"
      - "SketchFallback"
```

---

## 7. Enhanced Integration Testing Protocol

### 7.1. Multi-Component Integration Environment

```yaml
# e2e/kind/services.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: integration-scenarios
data:
  dual-write-config: |
    {
      "kafka": {
        "topics": {
          "raw": "proc.raw.v1",
          "sketch": "proc.sketch.v1"
        }
      },
      "redis": {
        "highWaterMarks": true
      }
    }
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kafka
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kafka
  template:
    metadata:
      labels:
        app: kafka
    spec:
      containers:
      - name: kafka
        image: confluentinc/cp-kafka:7.0.0
        ports:
        - containerPort: 9092
        env:
        - name: KAFKA_ADVERTISED_LISTENERS
          value: "PLAINTEXT://kafka:9092"
        - name: KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR
          value: "1"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7
        ports:
        - containerPort: 6379
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nrdb-stub
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nrdb-stub
  template:
    metadata:
      labels:
        app: nrdb-stub
    spec:
      containers:
      - name: nrdb-stub
        image: newrelic/nrdb-stub:latest
        ports:
        - containerPort: 8080
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sketch-merger
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sketch-merger
  template:
    metadata:
      labels:
        app: sketch-merger
    spec:
      containers:
      - name: sketch-merger
        image: newrelic/sketch-merger:latest
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: otelcol
spec:
  replicas: 1
  selector:
    matchLabels:
      app: otelcol
  template:
    metadata:
      labels:
        app: otelcol
    spec:
      containers:
      - name: otelcol
        image: otel/opentelemetry-collector:latest
```

### 7.2. Enhanced Integration Scenarios

For each implemented feature, run these enhanced integration scenarios:

1. **Multi-Platform Verification**:
   - Tests on Linux (multiple kernels)
   - Tests on Windows Server
   - Tests on macOS (when available)

2. **Dual-Write Validation**:
   - Verifies both raw and sketch data are correctly written
   - Confirms transaction integrity across both writes
   - Validates data consistency between formats
   - Ensures correct partitioning and ordering

3. **NRDB Storage Validation**:
   - Ensures data is correctly stored in NRDB
   - Validates schema compatibility
   - Confirms query correctness against stored data
   - Verifies compression and space efficiency

4. **Query Router Validation**:
   - Tests various NRQL query types
   - Verifies correct routing based on query type
   - Confirms fallback logic when sketch data is unavailable
   - Validates acceptable error bounds during fallback

5. **Failure Recovery**:
   - Tests agent behavior during service failures
   - Validates reconnection and backoff strategies
   - Confirms data buffers prevent data loss
   - Verifies diagnostic events during failures

### 7.3. Comprehensive Integration Report

Each integration test generates a detailed multi-section report:

```markdown
# Integration Test Report: topn_sampler

## Platform Compatibility
| Platform | Version | Status | Notes |
|----------|---------|--------|-------|
| Linux | 4.14 | ✅ Pass | |
| Linux | 5.10 | ✅ Pass | |
| Linux | 6.8 | ✅ Pass | |
| Windows | 2019 | ✅ Pass | |
| Windows | 2022 | ✅ Pass | |
| macOS | Latest | ✅ Pass | |

## Dual-Write Scenario
- ✅ Raw data written: 1250 events
- ✅ Sketch data written: 50 envelopes
- ✅ Transaction integrity: 100%
- ✅ Data equivalence: error margin 0.3%
- ✅ Partition validation: all partitions aligned

## NRDB Storage Scenario
- ✅ Schema validation: passed
- ✅ Data retention: 100%
- ✅ Query accuracy: p95 within 0.2% of expected
- ✅ Compression ratio: 10.2:1

## Query Router Scenario
- ✅ Standard NRQL routing: correct
- ✅ Percentile query routing: correct
- ✅ Fallback triggered successfully
- ✅ Raw data used for queries during fallback
- ✅ Error bound during fallback: 0.6%

## Failure Recovery
- ✅ Kafka restart: recovered in 3.2s
- ✅ OTLP collector restart: recovered in 2.1s
- ✅ No data loss during service restarts
- ✅ Backpressure correctly managed
- ✅ Diagnostic events properly emitted

## Blueprint Goal Coverage
| Goal | Status | Evidence |
|------|--------|----------|
| G-2 | ✅ Pass | CPU: 1.2%, Memory: 68MB |
| G-3 | ✅ Pass | p95/p99 error: 0.7% |
| G-4 | ✅ Pass | topn_capture_ratio: 97.3% |
| G-5 | ✅ Pass | Tail error: 3.2% |
| G-6 | ✅ Pass | Events emitted for all failures |
```

---

## 8. Enhanced Blueprint Alignment Verification

### 8.1. Comprehensive Milestone Tracking

Track progress against blueprint milestones with enhanced verification:

```bash
#!/bin/bash
# milestone_status.sh - Enhanced with verification checks

echo "# Blueprint Milestone Status with Verification"
echo "| Milestone | Description | Status | Date | Verification |"
echo "|-----------|-------------|--------|------|--------------|"

# Define milestones with verification commands
declare -A milestones=(
  ["M1"]="Process scanner refactored"
  ["M2"]="Top-N sampler implemented"
  ["M3"]="DDSketch integration complete"
  ["M4"]="OTLP exporter functional"
  ["M5"]="Watchdog module complete"
)

declare -A verifications=(
  ["M1"]="cd e2e && ./e2e_run.sh --scenario=process_scanner_perf"
  ["M2"]="cd e2e && ./e2e_run.sh --scenario=high_pid_churn"
  ["M3"]="cd e2e && ./e2e_run.sh --scenario=ddsketch_accuracy"
  ["M4"]="cd infrastructure-agent-testbed && ./test_otlp_export.sh"
  ["M5"]="cd e2e && ./e2e_run.sh --scenario=circuit_breaker"
)

# Check each milestone
for key in "${!milestones[@]}"; do
  status="⏳ PENDING"
  date="-"
  verification="Not run"
  
  # Check completion based on tasks in done/
  case "$key" in
    "M2")
      if [ -f "tasks/done/topn_sampler.yaml" ]; then
        status="✅ COMPLETE"
        date=$(stat -c %y "tasks/done/topn_sampler.yaml" | cut -d' ' -f1)
        
        # Run verification
        if eval "${verifications[$key]}"; then
          verification="✅ Verified"
        else
          verification="❌ Failed (see e2e/results/high_pid_churn)"
        fi
      fi
      ;;
    # Add checks for other milestones
  esac
  
  echo "| $key | ${milestones[$key]} | $status | $date | $verification |"
done
```

### 8.2. Enhanced Architecture Conformance Report

Generate comprehensive reports on architecture conformance:

```go
// architecture/conformance.go
package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ConformanceReport represents an architecture conformance report
type ConformanceReport struct {
	ModuleStructure     map[string]bool
	InterfaceAdherence  map[string]int
	PatternConformance  map[string]bool
	BoundaryViolations  []BoundaryViolation
	ResourceManagement  map[string]bool
}

// BoundaryViolation represents a module boundary violation
type BoundaryViolation struct {
	SourceModule string
	TargetModule string
	SourceFile   string
	LineNumber   int
	Description  string
}

// GenerateConformanceReport generates an architecture conformance report
func GenerateConformanceReport(rootDir string) (*ConformanceReport, error) {
	report := &ConformanceReport{
		ModuleStructure:    make(map[string]bool),
		InterfaceAdherence: make(map[string]int),
		PatternConformance: make(map[string]bool),
		BoundaryViolations: []BoundaryViolation{},
		ResourceManagement: make(map[string]bool),
	}
	
	// Verify module structure
	requiredModules := []string{"collector", "sampler", "sketch", "watchdog"}
	for _, module := range requiredModules {
		report.ModuleStructure[module] = checkModuleExists(rootDir, module)
	}
	
	// Check interface implementations
	requiredInterfaces := []string{"Resources()", "Shutdown()", "Init()"}
	for _, iface := range requiredInterfaces {
		report.InterfaceAdherence[iface] = countInterfaceImplementations(rootDir, iface)
	}
	
	// Validate architectural patterns
	patterns := []string{"circuit-breaker", "feature-flag", "lifecycle-hooks"}
	for _, pattern := range patterns {
		report.PatternConformance[pattern] = checkPatternImplemented(rootDir, pattern)
	}
	
	// Detect boundary violations
	report.BoundaryViolations = detectBoundaryViolations(rootDir)
	
	// Check resource management
	resources := []string{"memory-limit", "cpu-throttling", "file-descriptors"}
	for _, resource := range resources {
		report.ResourceManagement[resource] = checkResourceManagement(rootDir, resource)
	}
	
	return report, nil
}

// GenerateReport generates a Markdown report
func (r *ConformanceReport) GenerateReport() string {
	var b strings.Builder
	
	b.WriteString("# Architecture Conformance Report\n\n")
	
	// Module Structure
	b.WriteString("## Module Structure Conformance\n\n")
	for module, exists := range r.ModuleStructure {
		status := "✅ Present and correctly structured"
		if !exists {
			status = "❌ Missing or incorrectly structured"
		}
		b.WriteString(fmt.Sprintf("- %s: %s\n", module, status))
	}
	b.WriteString("\n")
	
	// Interface Adherence
	b.WriteString("## Interface Conformance\n\n")
	for iface, count := range r.InterfaceAdherence {
		status := "✅ Implemented across modules"
		if count < 4 {
			status = "❌ Missing implementations"
		}
		b.WriteString(fmt.Sprintf("- %s: %s (%d implementations)\n", iface, status, count))
	}
	b.WriteString("\n")
	
	// Add other sections...
	
	return b.String()
}
```

### 8.3. Deviation Management System

Enhanced system for tracking and validating necessary deviations:

```go
// deviation/manager.go
package deviation

import (
	"time"
)

// Deviation represents a deviation from the blueprint
type Deviation struct {
	ID           string
	Date         time.Time
	BlueprintItem string
	Change       string
	Justification string
	ImpactAnalysis map[string]string
	ApprovedBy    string
	Status        string
}

// Manager manages blueprint deviations
type Manager struct {
	deviations map[string]*Deviation
}

// NewManager creates a new deviation manager
func NewManager() *Manager {
	return &Manager{
		deviations: make(map[string]*Deviation),
	}
}

// RegisterDeviation registers a new deviation
func (m *Manager) RegisterDeviation(deviation *Deviation) error {
	// Validate deviation
	if err := validateDeviation(deviation); err != nil {
		return err
	}
	
	// Check if all non-negotiable goals are still met
	if err := verifyGoalsWithDeviation(deviation); err != nil {
		return err
	}
	
	// Register deviation
	m.deviations[deviation.ID] = deviation
	
	// Log deviation to file
	if err := logDeviation(deviation); err != nil {
		return err
	}
	
	return nil
}

// GetDeviation gets a deviation by ID
func (m *Manager) GetDeviation(id string) *Deviation {
	return m.deviations[id]
}

// ListDeviations lists all deviations
func (m *Manager) ListDeviations() []*Deviation {
	var result []*Deviation
	for _, d := range m.deviations {
		result = append(result, d)
	}
	return result
}

// GenerateReport generates a deviation report
func (m *Manager) GenerateReport() string {
	var b strings.Builder
	
	b.WriteString("# Blueprint Deviations\n\n")
	
	for _, d := range m.deviations {
		b.WriteString(fmt.Sprintf("## Deviation %s\n", d.ID))
		b.WriteString(fmt.Sprintf("- **Date**: %s\n", d.Date.Format("2006-01-02")))
		b.WriteString(fmt.Sprintf("- **Blueprint Item**: %s\n", d.BlueprintItem))
		b.WriteString(fmt.Sprintf("- **Change**: %s\n", d.Change))
		b.WriteString(fmt.Sprintf("- **Justification**: %s\n", d.Justification))
		b.WriteString("- **Impact Analysis**:\n")
		for goal, impact := range d.ImpactAnalysis {
			b.WriteString(fmt.Sprintf("  - %s: %s\n", goal, impact))
		}
		b.WriteString(fmt.Sprintf("- **Approved By**: %s\n", d.ApprovedBy))
		b.WriteString(fmt.Sprintf("- **Status**: %s\n", d.Status))
		b.WriteString("\n")
	}
	
	return b.String()
}
```

---

## 9. Enhanced Finishing Checklist

Before declaring implementation complete:

### 9.1. Feature Completeness

- [ ] All task files processed and moved to `tasks/done/`
- [ ] All blueprint milestones marked complete and verified
- [ ] All non-negotiable goals verified through `make verify`
- [ ] Multi-platform compatibility verified across Linux, Windows, and macOS
- [ ] All E2E test scenarios passing

### 9.2. Enhanced Stability Verification

- [ ] `make harness` stable green across 10 consecutive runs
- [ ] `make e2e` stable green across all platforms
- [ ] Resource usage consistently within thresholds
- [ ] No degradation under sustained load (24-hour test)
- [ ] All chaos scenarios properly handled
- [ ] Memory profiling shows no leaks over extended runs

### 9.3. Enhanced Documentation Completeness

- [ ] Runbook entries for all features, validated through scenarios
- [ ] CHANGELOG fully documents all changes
- [ ] Architecture documentation updated to reflect implementation
- [ ] Operational procedures documented for all failure scenarios
- [ ] Performance characteristics and limits clearly documented

### 9.4. Enhanced Final Goal Validation

Run a comprehensive verification of all goals G-1 through G-10 using the E2E framework:

```python
# validate_all_goals.py
#!/usr/bin/env python3
"""Validate all blueprint goals using the E2E framework."""

import os
import sys
import subprocess
import json
import datetime

def validate_goals():
    """Validate all blueprint goals."""
    goals = ["G-1", "G-2", "G-3", "G-4", "G-5", "G-6", "G-7", "G-8", "G-9", "G-10"]
    results = {}
    
    print("# Final Blueprint Goal Validation")
    print("| Goal | Description | Status | Evidence |")
    print("|------|-------------|--------|----------|")
    
    for goal in goals:
        description = get_goal_description(goal)
        
        # Run E2E validation for this goal
        result = run_goal_validation(goal)
        status = "✅ PASS" if result["passed"] else "❌ FAIL"
        evidence = result["evidence"]
        
        print(f"| {goal} | {description} | {status} | {evidence} |")
        
        results[goal] = {
            "description": description,
            "passed": result["passed"],
            "evidence": evidence
        }
    
    # Save results
    with open("final_validation_results.json", "w") as f:
        json.dump({
            "timestamp": datetime.datetime.now().isoformat(),
            "results": results,
            "overall_passed": all(r["passed"] for r in results.values())
        }, f, indent=2)
    
    # Overall result
    overall = all(r["passed"] for r in results.values())
    print("\nOverall result:", "✅ PASS" if overall else "❌ FAIL")
    
    return overall

def get_goal_description(goal):
    """Get the description for a goal."""
    descriptions = {
        "G-1": "Additive-Only Schema",
        "G-2": "Host Safety",
        "G-3": "Statistical Fidelity",
        "G-4": "Top-N Accuracy",
        "G-5": "Tail Error Bound",
        "G-6": "Self-Governance",
        "G-7": "Documentation Parity",
        "G-8": "Local Test Green",
        "G-9": "Architectural Conformance",
        "G-10": "Documentation Quality"
    }
    return descriptions.get(goal, "Unknown Goal")

def run_goal_validation(goal):
    """Run E2E validation for a specific goal."""
    scenario_mapping = {
        "G-1": "proto_evolution",
        "G-2": "high_pid_churn",
        "G-3": "ddsketch_accuracy",
        "G-4": "high_pid_churn",
        "G-5": "tail_aggregation",
        "G-6": "failover",
        "G-7": "runbook_scenarios",
        "G-8": "all_tests",
        "G-9": "architecture_check",
        "G-10": "runbook_scenarios"
    }
    
    scenario = scenario_mapping.get(goal, "standard")
    
    try:
        # Run the scenario validation
        result = subprocess.run(
            ["./e2e/e2e_run.sh", f"--scenario={scenario}", f"--goal={goal}"],
            capture_output=True, text=True, check=False
        )
        
        if result.returncode == 0:
            # Extract evidence from output
            evidence_line = next((line for line in result.stdout.splitlines() if f"[{goal}]" in line), "")
            evidence = evidence_line.split("[EVIDENCE]")[-1].strip() if "[EVIDENCE]" in evidence_line else "Passed"
            return {"passed": True, "evidence": evidence}
        else:
            # Extract failure details
            failure_line = next((line for line in result.stderr.splitlines() if f"[{goal}]" in line), "")
            evidence = failure_line.split("[FAILURE]")[-1].strip() if "[FAILURE]" in failure_line else "Failed"
            return {"passed": False, "evidence": evidence}
    except Exception as e:
        return {"passed": False, "evidence": f"Error: {str(e)}"}

if __name__ == "__main__":
    success = validate_goals()
    sys.exit(0 if success else 1)
```

### 9.5. Enhanced Final Deliverables

- [ ] Compiled agent binary with all features for all platforms
- [ ] Complete documentation package
- [ ] E2E test framework with multi-platform support
- [ ] Comprehensive test results across all platforms
- [ ] Performance benchmark results with comparison to baseline
- [ ] Architecture conformance report
- [ ] Chaos test results demonstrating resilience
- [ ] Blueprint goal validation report

---

## 10. CI/CD Integration

### 10.1. Enhanced CI/CD Pipeline

```yaml
# .github/workflows/ci.yml
name: CI/CD Pipeline

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  static-analysis:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      - name: Lint
        run: make lint
      - name: Proto compatibility check
        run: cd infrastructure-agent && buf breaking --against 'HEAD^'
  
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      - name: Run unit tests
        run: make test
      - name: Upload coverage
        uses: actions/upload-artifact@v3
        with:
          name: coverage
          path: infrastructure-agent/coverage.out
  
  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Set up environment
        run: docker compose up -d
      - name: Run integration tests
        run: make integration
      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: integration-results
          path: infrastructure-agent-testbed/results
  
  e2e-tests:
    strategy:
      matrix:
        platform: [linux, windows]
        scenario: [standard, high_pid_churn]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Set up kind cluster
        uses: helm/kind-action@v1.5.0
      - name: Deploy services
        run: kubectl apply -f e2e/kind/services.yaml
      - name: Run E2E tests
        run: cd e2e && ./e2e_run.sh --scenario=${{ matrix.scenario }} --platform=${{ matrix.platform }}
      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: e2e-${{ matrix.platform }}-${{ matrix.scenario }}
          path: e2e/results
  
  blueprint-validation:
    needs: [e2e-tests]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Download all results
        uses: actions/download-artifact@v3
        with:
          path: artifacts
      - name: Validate blueprint goals
        run: python3 e2e/validate_all_goals.py
      - name: Upload validation report
        uses: actions/upload-artifact@v3
        with:
          name: blueprint-validation
          path: final_validation_results.json
  
  publish:
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    needs: [static-analysis, unit-tests, integration-tests, e2e-tests, blueprint-validation]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Build binaries
        run: make build
      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: binaries
          path: infrastructure-agent/dist
```

---

*(End of Enhanced AI-Infrastructure Agent Implementation Plan with E2E Testing Framework)*
