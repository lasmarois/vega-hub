#!/bin/bash
#
# vega-hub-ask.sh - PreToolUse hook for AskUserQuestion
#
# This hook intercepts AskUserQuestion tool calls and routes them to vega-hub.
# It blocks until a human answers via the vega-hub web UI.
#
# Input: JSON from Claude Code PreToolUse hook (via stdin)
# Output: JSON with permissionDecision: deny and answer in permissionDecisionReason
#

set -euo pipefail

# Read hook input from stdin
INPUT=$(cat)

# Extract tool name - only process AskUserQuestion
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty')
if [[ "$TOOL_NAME" != "AskUserQuestion" ]]; then
    # Not our tool, let it proceed normally
    exit 0
fi

# Get vega-hub port from environment or config
VEGA_HUB_PORT="${VEGA_HUB_PORT:-8080}"
VEGA_HUB_HOST="${VEGA_HUB_HOST:-localhost}"

# Extract goal ID from cwd (worktree path like .../goal-10-add-auth)
CWD=$(echo "$INPUT" | jq -r '.cwd // empty')
GOAL_ID=$(basename "$CWD" | grep -oP 'goal-\K\d+' || echo "0")

# Extract session ID
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // "unknown"')

# Extract question details from tool_input
TOOL_INPUT=$(echo "$INPUT" | jq -c '.tool_input // {}')

# Build the first question (AskUserQuestion can have multiple, we take the first)
QUESTION=$(echo "$TOOL_INPUT" | jq -r '.questions[0].question // empty')
OPTIONS=$(echo "$TOOL_INPUT" | jq -c '.questions[0].options // []')

if [[ -z "$QUESTION" ]]; then
    # No question found, let it proceed
    exit 0
fi

# Build request for vega-hub
REQUEST=$(jq -n \
    --argjson goal_id "$GOAL_ID" \
    --arg session_id "$SESSION_ID" \
    --arg question "$QUESTION" \
    --argjson options "$OPTIONS" \
    '{
        goal_id: $goal_id,
        session_id: $session_id,
        question: $question,
        options: $options
    }')

# POST to vega-hub (blocks until answered)
RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "$REQUEST" \
    "http://${VEGA_HUB_HOST}:${VEGA_HUB_PORT}/api/ask" \
    2>/dev/null) || {
    # vega-hub not available, let tool proceed normally
    echo "Warning: vega-hub not available at $VEGA_HUB_HOST:$VEGA_HUB_PORT" >&2
    exit 0
}

# Extract answer
ANSWER=$(echo "$RESPONSE" | jq -r '.answer // empty')

if [[ -z "$ANSWER" ]]; then
    # No answer received, let it proceed
    exit 0
fi

# Build hook response - deny the tool but provide the answer
cat <<EOF
{
    "hookSpecificOutput": {
        "hookEventName": "PreToolUse",
        "permissionDecision": "deny",
        "permissionDecisionReason": "[vega-hub] User answered your question. Response: '$ANSWER'. Continue with this information."
    }
}
EOF

exit 0
