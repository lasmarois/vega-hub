#!/usr/bin/env bash
#
# on-stop.sh - Hook that runs when executor stops
#
# This hook notifies vega-hub that the executor has stopped.
# vega-hub handles: markdown updates, desktop notifications, SSE events.
# All communication goes through vega-hub (single source of truth).
#
# Input: JSON via stdin with session info
# Output: JSON allowing stop (never blocks)

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
    # Not in a goal worktree, nothing to report
    exit 0
fi

GOAL_ID="${BASH_REMATCH[1]}"

# Get vega-hub settings
VEGA_HUB_PORT="${VEGA_HUB_PORT:-8080}"
VEGA_HUB_HOST="${VEGA_HUB_HOST:-localhost}"

# Best effort: notify vega-hub (don't fail if unavailable)
notify_vega_hub() {
    # Build request
    local request
    request=$(jq -n \
        --argjson goal_id "$GOAL_ID" \
        --arg session_id "$SESSION_ID" \
        --arg reason "completed" \
        '{
            goal_id: $goal_id,
            session_id: $session_id,
            reason: $reason
        }')

    # POST to vega-hub (fire and forget)
    curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$request" \
        "http://${VEGA_HUB_HOST}:${VEGA_HUB_PORT}/api/executor/stop" \
        >/dev/null 2>&1 || true
}

# Notify (best effort)
notify_vega_hub 2>/dev/null || true

# Always allow stop - never block
echo '{"decision": null}'
exit 0
