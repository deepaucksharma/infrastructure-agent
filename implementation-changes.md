# AI-Infrastructure Agent Implementation Changes

## Agent-Side Mandate — What Actually Runs on the Host

| **Module** | **New Code?** | **Core Function** | **Key Constraints** | **Relevant Files** |
|------------|---------------|-------------------|---------------------|-------------------|
| **Process Scanner** | *Refactor* | Walk /proc, compute cpu%, rss | ≤ 1.5 % host CPU | [collector/process_scanner.go](./collector/process_scanner.go), [collector/process_info.go](./collector/process_info.go) |
| **Top-N Heap** | **NEW** | Maintain N highest-score PIDs (default 20) | O(k log N); bypass if *k ≤ 1.25 N* | [sampler/heap.go](./sampler/heap.go), [sampler/topn.go](./sampler/topn.go) |
| **Tail Aggregator** | **NEW** | Group all remaining PIDs by processDisplayName, emit ProcessAggregateSample | Error ≤ 5 % on fleet CPU/RSS totals | [sampler/sampler.go](./sampler/sampler.go) |
| **DDSketch Encoder** | **NEW** | Build DDSketch (γ = 0.0075) per metric, wrap in SketchEnvelope v1 | ≤ 1 % p95/p99 rel. error | [sketch/ddsketch.go](./sketch/ddsketch.go), [sketch/serialization.go](./sketch/serialization.go) |
| **WASM Filter Runtime** | *Optional* | Load NR-signed plugins for truncation/redaction | Wasmtime-go, fuel 10 ms, 4 MiB cap | *Planned for future implementation* |
| **eBPF Lifecycle Tracker** | *Optional* | Capture < 5 s exec/exit events, summarise to ProcessEphemeralSummary | Lost events < 0.1 %; auto-fallback | *Planned for future implementation* |
| **Health & Circuit Breakers** | **NEW** | Expose agent.* metrics + AgentDiagEvent for every trip or fallback | Never silently drop data | [watchdog/circuit_breaker.go](./watchdog/circuit_breaker.go), [watchdog/component_monitor.go](./watchdog/component_monitor.go) |
| **Hot-Reload Config** | *Extend* | All keys (top_n, adaptive_n, sketch_enable, etc.) patchable by Fleet API | Change takes effect < 30 s | *Extended implementation in progress* |

## Wire-Contracts the Agent Must Honour

| **Envelope** | **Purpose** | **Immutable Fields** | **Relevant Files** |
|--------------|-------------|----------------------|-------------------|
| ProcessEnvelope v2.1 | Top-N PID rows **and** aggregated tail rows | entity_guid, schema_version, payload (TopNBatch) | [sampler/topn.go](./sampler/topn.go) |
| SketchEnvelope v1 | DDSketch payload for each metric × key | metric_name, gamma, sketch_version, histogram | [sketch/serialization.go](./sketch/serialization.go) |
| AgentDiagEvent | Health / breaker telemetry | module, event_type, reason, ts_ns | [watchdog/circuit_breaker.go](./watchdog/circuit_breaker.go) |

*(Protobuf tags ≤ 19 frozen; every additive change bumps the schema_version string.)*

## Minimal Supporting Changes Outside the Agent

| **Area** | **Why Needed** | **Exact Scope ("no more, no less")** |
|----------|---------------|-------------------------------------|
| **OTLP Receiver** | Accept SketchEnvelope over gRPC; enforce TLS + 2 000-attr cap | • proto decode extension • 429 back-pressure |
| **Kafka Topics** | Split raw vs. sketch for dual-write | proc.raw.v1, proc.sketch.v1 (same partition strategy) |
| **Sketch Merger** | Merge envelopes per 60 s window, output DistributionMetric | • Version/gamma guard • RocksDB state TTL |
| **NRDB Hot Path** | Store ProcessSample, ProcessAggregateSample rows like today | *No schema change required* |
| **NRDB Sketch Store (v2)** | Column family for DistributionMetric blobs | Compression Zstd-3 |
| **Query-Compatibility Plugin** | Classify, rewrite, route NRQL (approxPercentile) | Latency ≤ 5 ms; fallback RAW if sketch gap |
| **Redis High-Water Mark** | Tell router when sketch data is complete | Single key per account: highest ingest_timestamp dual-written |

*No FinOps dashboards, no ML, no sparse-parquet—those can wait; the above is the minimum contract for the new agent to function and for customers to stay unbroken.*

## Agent Build & Test Checklist

- [x] topn_capture_ratio metric exported (Prom & OTEL) - [sampler/metrics.go](./sampler/metrics.go)
- [x] Heap bypass verified (< 50 PIDs) - [sampler/heap_test.go](./sampler/heap_test.go)
- [x] DDSketch → OTLP mapping unit-tests (100 random distributions) - [sketch/ddsketch_test.go](./sketch/ddsketch_test.go)
- [x] Replay harness: raw vs. smart+sketch ≤ 1 % quantile error - [../infrastructure-agent-testbed/run_replay.sh](../infrastructure-agent-testbed/run_replay.sh)
- [x] Circuit-breaker trips produce AgentDiagEvent and drop to RAW - [watchdog/tests](./watchdog/tests)
- [x] All new config keys documented in docs/agent-config.md - [docs/agent-config.md](./docs/agent-config.md)

## Roll-Out Sequence

| **Week** | **Action** | **Success Gate** |
|----------|------------|------------------|
| **W-0** | Ship agent with mode=smart **disabled** | Canary CPU + RSS delta ≤ +3 % |
| **W-2** | Enable smart default on 5 % fleet | topn_capture_ratio ≥ 95 % in telemetry |
| **W-4** | Enable sketch_enable=true on same canaries | Validation harness diff ≤ 1 % |
| **W-6** | Deploy Sketch Merger, turn on **dual-write** | Redis high-water mark matches ingest Ts |
| **W-8** | Flip router default to **SKETCH** for percentile queries | p95 latency drop 30 %; zero dashboard breakage |

## Outcome at GA

* **Series cut by ~50 %** (Top-N + aggregation)
* **Wire bytes cut by ~90 %** (DDSketch vs. raw)
* **p95/p99 accuracy ≤ 1 %**, anomaly recall ≥ 95 %
* Agent stays within **≤ 2 % CPU / 100 MB RSS**

Ship this, and the infra agent stops being a cardinality liability—becomes a lean, self-optimising telemetry source that the rest of the pipeline can digest without heroic tuning.
