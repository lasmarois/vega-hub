#!/usr/bin/env bash
#
# on-session-start.sh - Hook that runs when executor session starts
#
# NOTE: When executors are spawned via vega-hub API (spawn.go), registration
# is already handled by spawn.go. This hook should NOT re-register, as that
# causes duplicate executor entries.
#
# This hook only provides context for sessions NOT spawned via vega-hub.
#
# Input: JSON via stdin with session info
# Output: JSON with additionalContext (if in goal worktree)

set -euo pipefail

# Read input
INPUT=$(cat)

# Get session info
CWD=$(echo "$INPUT" | jq -r '.cwd // empty')

if [[ -z "$CWD" ]]; then
    exit 0
fi

# Try to detect goal from directory name (pattern: goal-N-slug)
GOAL_DIR=$(basename "$CWD")
if [[ ! "$GOAL_DIR" =~ ^goal-([0-9]+)- ]]; then
    # Not in a goal worktree, nothing to do
    exit 0
fi

GOAL_ID="${BASH_REMATCH[1]}"

# Build context directly (don't call vega-hub register endpoint)
# spawn.go already registered us if we were spawned via API
CONTEXT="[EXECUTOR SESSION START]
Working on Goal #${GOAL_ID}
Directory: ${CWD}

IMPORTANT REMINDERS:
1. Load 'planning-with-files' skill if not already loaded
2. Planning files go at worktree root: task_plan.md, findings.md, progress.md
3. You can use AskUserQuestion to ask the human questions directly (via vega-hub)
4. Before completing, you MUST:
   - Archive planning files to docs/planning/history/goal-${GOAL_ID}/
   - Commit the archive
   - Report to manager for approval
5. Commit messages must include 'Goal: #${GOAL_ID}'"

# Output JSON with context for Claude
cat <<EOF
{
    "hookSpecificOutput": {
        "hookEventName": "SessionStart",
        "additionalContext": $(echo "$CONTEXT" | jq -Rs .)
    }
}
EOF

exit 0
