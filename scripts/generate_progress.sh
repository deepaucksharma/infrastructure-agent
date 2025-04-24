#!/bin/bash
set -e

echo "# Progress Tracking Dashboard"
echo
echo "## Task Completion Status"
echo
echo "| Task | Phase | Status | Acceptance Criteria Met |"
echo "|------|-------|--------|-------------------------|"

# Count tasks in each directory
TODO_COUNT=$(ls -1 ../tasks/*.yaml 2>/dev/null | wc -l)
DONE_COUNT=$(ls -1 ../tasks/done/*.yaml 2>/dev/null | wc -l)
TOTAL_COUNT=$((TODO_COUNT + DONE_COUNT))

# List tasks from todo
for task in ../tasks/*.yaml; do
  if [ -f "$task" ]; then
    slug=$(grep "slug:" "$task" | cut -d: -f2 | tr -d ' ')
    phase=$(grep "phase:" "$task" | cut -d: -f2 | tr -d ' ')
    echo "| $slug | $phase | TODO | - |"
  fi
done

# List tasks from done
for task in ../tasks/done/*.yaml; do
  if [ -f "$task" ]; then
    slug=$(grep "slug:" "$task" | cut -d: -f2 | tr -d ' ')
    phase=$(grep "phase:" "$task" | cut -d: -f2 | tr -d ' ')
    echo "| $slug | $phase | DONE | ✅ |"
  fi
done

# Overall progress
echo
echo "## Overall Progress"
echo
echo "* Total tasks: $TOTAL_COUNT"
echo "* Completed: $DONE_COUNT"
echo "* Remaining: $TODO_COUNT"
echo "* Progress: $(($DONE_COUNT * 100 / $TOTAL_COUNT))%"

# ASCII progress bar
progress_width=$(($DONE_COUNT * 50 / $TOTAL_COUNT))
echo
echo "\`\`\`"
echo -n "["
for ((i=0; i<progress_width; i++)); do echo -n "#"; done
for ((i=progress_width; i<50; i++)); do echo -n " "; done
echo "] $(($DONE_COUNT * 100 / $TOTAL_COUNT))%"
echo "\`\`\`"
