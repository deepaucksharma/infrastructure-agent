# Final Verification Checklist

## Epic Completion
- [ ] All epics from blueprint completed
- [ ] All stories implemented

## Goal Verification
- [ ] G-1: Additive-Only Schema - `buf breaking` returns 0
- [ ] G-2: Host Safety - CPU ≤ 2%, RSS ≤ 100 MB on 500-PID replay run
- [ ] G-3: Statistical Fidelity - DDSketch p95/p99 error ≤ 1% (γ = 0.0075)
- [ ] G-4: Top-N Accuracy - `topn_capture_ratio` ≥ 95%
- [ ] G-5: Tail Error Bound - Aggregated CPU/RSS sums Δ ≤ 5%
- [ ] G-6: Self-Governance - Breaker/fallback emits `AgentDiagEvent`
- [ ] G-7: Documentation Parity - All flags/metrics/events documented
- [ ] G-8: Local Tests Green - `make lint test bench harness` all exit 0
- [ ] G-9: Architectural Conformance - All tasks include arch review
- [ ] G-10: Doc Quality Validation - Runbook validated with scenario

## Integration Testing
- [ ] 10 consecutive green harness runs
- [ ] Integration tests pass
- [ ] Runbook validated

## Documentation
- [ ] All new features documented
- [ ] Runbook complete
- [ ] API reference updated
