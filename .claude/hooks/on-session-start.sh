#!/usr/bin/env bash
#
# on-session-start.sh - Hook that runs when executor session starts
#
# NOTE: When executors are spawned via vega-hub API (spawn.go), registration
# is already handled by spawn.go. This hook should NOT re-register, as that
# causes duplicate executor entries.
#
# This hook:
# 1. Checks if planning-with-files skill is installed
# 2. Detects the goal being worked on (from worktree directory name)
# 3. Injects context about the goal into the session
# 4. Reminds executor of key workflow requirements
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

# Function to check if planning-with-files skill is installed
check_planning_skill() {
    local plugin_cache="$HOME/.claude/plugins/cache/planning-with-files"

    if [[ -d "$plugin_cache" ]] && [[ -n "$(ls -A "$plugin_cache" 2>/dev/null)" ]]; then
        return 0  # Skill is installed
    else
        return 1  # Skill is NOT installed
    fi
}

# Try to detect goal from directory name (pattern: goal-N-slug or goal-HASH-slug)
GOAL_DIR=$(basename "$CWD")
if [[ "$GOAL_DIR" =~ ^goal-([0-9a-f]+)- ]]; then
    GOAL_ID="${BASH_REMATCH[1]}"
else
    # Not in a goal worktree, nothing to do
    exit 0
fi

# Check for planning-with-files skill
SKILL_WARNING=""
if ! check_planning_skill; then
    SKILL_WARNING="
**CRITICAL: planning-with-files skill NOT INSTALLED**

The planning-with-files skill is REQUIRED for vega-missile executors.
Without it, you cannot properly track your work.

To install, run:
  claude plugins install OthmanAdi/planning-with-files

Then restart this session.

Source: https://github.com/OthmanAdi/planning-with-files

"
fi

# Build context directly (don't call vega-hub register endpoint)
# spawn.go already registered us if we were spawned via API
CONTEXT="[EXECUTOR SESSION START]
Working on Goal #${GOAL_ID}
Directory: ${CWD}
${SKILL_WARNING}
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
