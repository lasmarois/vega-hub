#!/usr/bin/env bash
#
# on-session-start.sh - Hook that runs when executor session starts
#
# This hook registers the executor with vega-hub and receives context.
# All communication goes through vega-hub (single source of truth).
#
# Input: JSON via stdin with session info
# Output: JSON with additionalContext from vega-hub

set -euo pipefail

# Read input
INPUT=$(cat)

# Get session info
CWD=$(echo "$INPUT" | jq -r '.cwd // empty')
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // "unknown"')

if [[ -z "$CWD" ]]; then
    exit 0
fi

# Try to detect goal from directory name (pattern: goal-N-slug)
GOAL_DIR=$(basename "$CWD")
if [[ ! "$GOAL_DIR" =~ ^goal-([0-9]+)- ]]; then
    # Not in a goal worktree, nothing to register
    exit 0
fi

GOAL_ID="${BASH_REMATCH[1]}"

# Get vega-hub settings
VEGA_HUB_PORT="${VEGA_HUB_PORT:-8080}"
VEGA_HUB_HOST="${VEGA_HUB_HOST:-localhost}"

# Check if vega-hub is running
if ! curl -s "http://${VEGA_HUB_HOST}:${VEGA_HUB_PORT}/api/health" >/dev/null 2>&1; then
    # vega-hub not available, exit silently
    exit 0
fi

# Build request
REQUEST=$(jq -n \
    --argjson goal_id "$GOAL_ID" \
    --arg session_id "$SESSION_ID" \
    --arg cwd "$CWD" \
    '{
        goal_id: $goal_id,
        session_id: $session_id,
        cwd: $cwd
    }')

# POST to vega-hub
RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "$REQUEST" \
    "http://${VEGA_HUB_HOST}:${VEGA_HUB_PORT}/api/executor/register" \
    2>/dev/null) || {
    # Failed to register, exit silently
    exit 0
}

# Extract context from response
CONTEXT=$(echo "$RESPONSE" | jq -r '.context // empty')

if [[ -z "$CONTEXT" ]]; then
    exit 0
fi

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
