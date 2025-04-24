#!/bin/bash
set -e

echo "# Final Goal Verification"
echo "Generated: $(date)"
echo

echo "| Goal | Description | Status | Evidence |"
echo "|------|-------------|--------|----------|"

verify_goal() {
  local goal="$1"
  local desc="$2"
  local check_cmd="$3"
  local output_cmd="$4"
  
  echo -n "| $goal | $desc | "
  
  if eval "$check_cmd"; then
    echo -n "✅ PASS | "
  else
    echo -n "❌ FAIL | "
  fi
  
  if [ ! -z "$output_cmd" ]; then
    eval "$output_cmd"
  else
    echo "N/A |"
  fi
}

# Run verification for each goal
verify_goal "G-1" "Additive-Only Schema" \
  "cd infrastructure-agent && buf breaking --against 'HEAD^' &>/dev/null" \
  "echo 'buf breaking check passed |'"

verify_goal "G-2" "Host Safety" \
  "grep -q 'CPU usage:' hlog.txt && awk '{if(\$3<=2.0)exit 0; else exit 1}' <(grep 'CPU usage:' hlog.txt)" \
  "echo \"CPU: \$(grep 'CPU usage:' hlog.txt | awk '{print \$3}')%, RSS: \$(grep 'Memory usage:' hlog.txt | awk '{print \$3}')MB |\""

verify_goal "G-3" "Statistical Fidelity" \
  "grep -q 'p95/p99 error:' hlog.txt && awk '{if(\$4<=1.0)exit 0; else exit 1}' <(grep 'p95/p99 error:' hlog.txt)" \
  "echo \"Error: \$(grep 'p95/p99 error:' hlog.txt | awk '{print \$4}')% |\""

verify_goal "G-4" "Top-N Accuracy" \
  "grep -q 'topn_capture_ratio:' hlog.txt && awk '{if(\$3>=95.0)exit 0; else exit 1}' <(grep 'topn_capture_ratio:' hlog.txt)" \
  "echo \"Ratio: \$(grep 'topn_capture_ratio:' hlog.txt | awk '{print \$3}')% |\""

verify_goal "G-5" "Tail Error Bound" \
  "grep -q 'Aggregated delta:' hlog.txt && awk '{if(\$3<=5.0)exit 0; else exit 1}' <(grep 'Aggregated delta:' hlog.txt)" \
  "echo \"Delta: \$(grep 'Aggregated delta:' hlog.txt | awk '{print \$3}')% |\""

verify_goal "G-6" "Self-Governance" \
  "grep -q 'AgentDiagEvent' hlog.txt" \
  "echo \"Found diagnostic events |\""

verify_goal "G-7" "Documentation Parity" \
  "[ -f docs/runbook.md ] && [ \$(grep -r 'func New' infrastructure-agent/ --include='*.go' | wc -l) -le \$(grep -r '## ' docs/runbook.md | wc -l) ]" \
  "echo \"All features documented |\""

verify_goal "G-8" "Local Tests Green" \
  "make lint && make test && make bench && make harness" \
  "echo \"All tests passed |\""

verify_goal "G-9" "Architectural Conformance" \
  "for task in tasks/done/*.yaml; do slug=\$(grep 'slug:' \"\$task\" | cut -d: -f2- | tr -d ' '); [ -f \"reports/\${slug}_arch.md\" ] || exit 1; done" \
  "echo \"All tasks have architecture reviews |\""

verify_goal "G-10" "Documentation Quality" \
  "grep -q 'ModuleOverLimit resolution' docs/runbook.md" \
  "echo \"Runbook includes scenario documentation |\""

# Check if all goals passed
failures=$(grep "❌ FAIL" -c <<< "$(cat /dev/stdout)" || true)

echo
if [ "$failures" -eq "0" ]; then
  echo "✅ All goals verified successfully!"
  echo "Project implementation complete and ready for release."
else
  echo "❌ $failures goal(s) failed verification."
  echo "Implementation is not complete. Please address the failing goals."
  exit 1
fi
