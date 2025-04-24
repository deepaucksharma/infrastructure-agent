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
if grep -q "CPU usage:" hlog.txt && grep -q "Memory usage:" hlog.txt; then
  cpu=$(grep "CPU usage:" hlog.txt | awk '{print $3}')
  mem=$(grep "Memory usage:" hlog.txt | awk '{print $3}')
  
  if (( $(echo "$cpu <= 2.0" | bc -l) )) && (( $(echo "$mem <= 100" | bc -l) )); then
    echo "✅ G-2 PASS: CPU ($cpu%) and Memory ($mem MB) within limits"
  else
    echo "❌ G-2 FAIL: Resource usage exceeds limits (CPU: $cpu%, Memory: $mem MB)"
    exit 1
  fi
fi

# G-3: Statistical Fidelity
if grep -q "p95/p99 error:" hlog.txt; then
  error=$(grep "p95/p99 error:" hlog.txt | awk '{print $4}')
  
  if (( $(echo "$error <= 1.0" | bc -l) )); then
    echo "✅ G-3 PASS: Statistical error ($error%) within bounds"
  else
    echo "❌ G-3 FAIL: Statistical error ($error%) exceeds 1%"
    exit 1
  fi
fi

# G-4: Top-N Accuracy
if grep -q "topn_capture_ratio:" hlog.txt; then
  ratio=$(grep "topn_capture_ratio:" hlog.txt | awk '{print $3}')
  
  if (( $(echo "$ratio >= 95.0" | bc -l) )); then
    echo "✅ G-4 PASS: Top-N capture ratio ($ratio%) meets threshold"
  else
    echo "❌ G-4 FAIL: Top-N capture ratio ($ratio%) below 95%"
    exit 1
  fi
fi

# G-5: Tail Error Bound
if grep -q "Aggregated delta:" hlog.txt; then
  delta=$(grep "Aggregated delta:" hlog.txt | awk '{print $3}')
  
  if (( $(echo "$delta <= 5.0" | bc -l) )); then
    echo "✅ G-5 PASS: Aggregated delta ($delta%) within bounds"
  else
    echo "❌ G-5 FAIL: Aggregated delta ($delta%) exceeds 5%"
    exit 1
  fi
fi

# G-6: Self-Governance
if grep -q "AgentDiagEvent" hlog.txt; then
  echo "✅ G-6 PASS: Agent diagnostic events present"
else
  echo "❌ G-6 FAIL: No agent diagnostic events found"
  exit 1
fi

# G-7: Documentation Parity
if [ -f docs/runbook.md ]; then
  # Count features in code vs. runbook
  code_features=$(grep -r "func New" infrastructure-agent/ --include="*.go" | wc -l)
  doc_features=$(grep -r "## " docs/runbook.md | wc -l)
  
  if [ $doc_features -ge $code_features ]; then
    echo "✅ G-7 PASS: Documentation covers all features"
  else
    echo "❌ G-7 FAIL: Documentation missing for some features"
    exit 1
  fi
fi

# G-8: Local Tests Green
make lint && make test && make bench && make harness
if [ $? -eq 0 ]; then
  echo "✅ G-8 PASS: All local tests passed"
else
  echo "❌ G-8 FAIL: Some tests failed"
  exit 1
fi

# G-9: Architectural Conformance
for task in tasks/done/*.yaml; do
  slug=$(grep "slug:" "$task" | cut -d: -f2- | tr -d ' ')
  if [ ! -f "reports/${slug}_arch.md" ]; then
    echo "❌ G-9 FAIL: Missing architecture review for $slug"
    exit 1
  fi
done
echo "✅ G-9 PASS: Architecture reviews present for all tasks"

# G-10: Documentation Quality
if grep -q "ModuleOverLimit resolution" docs/runbook.md; then
  echo "✅ G-10 PASS: Runbook includes scenario resolution guidance"
else
  echo "❌ G-10 FAIL: Runbook missing scenario resolution guidance"
  exit 1
fi

echo "All goals G-1 through G-10 PASSED!"
