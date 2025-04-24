#!/bin/bash
set -e

echo "# Blueprint Milestone Status"
echo "Generated: $(date)"
echo

# Define milestones from blueprint
declare -A milestones=(
  ["M1"]="Process scanner refactored"
  ["M2"]="Top-N sampler implemented"
  ["M3"]="DDSketch integration complete"
  ["M4"]="OTLP exporter functional"
  ["M5"]="Watchdog module complete"
)

echo "| Milestone | Description | Status | Date |"
echo "|-----------|-------------|--------|------|"

# Check each milestone
for key in "${!milestones[@]}"; do
  status="⏳ PENDING"
  date="-"
  
  # Check completion based on tasks in done/
  case "$key" in
    "M1")
      if [ -f "tasks/done/process_scanner.yaml" ]; then
        status="✅ COMPLETE"
        date=$(stat -c %y "tasks/done/process_scanner.yaml" | cut -d' ' -f1)
      fi
      ;;
    "M2")
      if [ -f "tasks/done/topn_sampler.yaml" ]; then
        status="✅ COMPLETE"
        date=$(stat -c %y "tasks/done/topn_sampler.yaml" | cut -d' ' -f1)
      fi
      ;;
    "M3")
      if [ -f "tasks/done/ddsketch_impl.yaml" ]; then
        status="✅ COMPLETE"
        date=$(stat -c %y "tasks/done/ddsketch_impl.yaml" | cut -d' ' -f1)
      fi
      ;;
    "M4")
      if [ -f "tasks/done/otlp_exporter.yaml" ]; then
        status="✅ COMPLETE"
        date=$(stat -c %y "tasks/done/otlp_exporter.yaml" | cut -d' ' -f1)
      fi
      ;;
    "M5")
      if [ -f "tasks/done/watchdog_module.yaml" ]; then
        status="✅ COMPLETE"
        date=$(stat -c %y "tasks/done/watchdog_module.yaml" | cut -d' ' -f1)
      fi
      ;;
  esac
  
  echo "| $key | ${milestones[$key]} | $status | $date |"
done

# Calculate overall progress
total=5
completed=0
for key in "${!milestones[@]}"; do
  case "$key" in
    "M1")
      [ -f "tasks/done/process_scanner.yaml" ] && completed=$((completed+1))
      ;;
    "M2")
      [ -f "tasks/done/topn_sampler.yaml" ] && completed=$((completed+1))
      ;;
    "M3")
      [ -f "tasks/done/ddsketch_impl.yaml" ] && completed=$((completed+1))
      ;;
    "M4")
      [ -f "tasks/done/otlp_exporter.yaml" ] && completed=$((completed+1))
      ;;
    "M5")
      [ -f "tasks/done/watchdog_module.yaml" ] && completed=$((completed+1))
      ;;
  esac
done

echo
echo "## Overall Progress"
echo "* Completed: $completed of $total milestones"
echo "* Progress: $((completed * 100 / total))%"

# Overall project status
if [ $completed -eq $total ]; then
  status="✅ COMPLETED"
elif [ $completed -eq 0 ]; then
  status="⏳ NOT STARTED"
else
  status="🔄 IN PROGRESS"
fi

echo "## Project Status: $status"
