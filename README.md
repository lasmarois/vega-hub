# vega-hub

Real-time communication hub for human-executor interaction in vega-missile.

## Overview

vega-hub enables direct communication between humans and Claude Code executors via:
- Hook interception of `AskUserQuestion` tool calls
- Web UI for answering questions in real-time
- Markdown persistence for Q&A history

## Quick Start

### Build

```bash
make build
```

This creates `./dist/vega-hub` - a single binary with embedded React UI.

### Development

```bash
# Start frontend (hot reload) and backend (hot reload)
make dev
```

- Frontend: http://localhost:5173
- Backend API: http://localhost:8080

### Run

```bash
./dist/vega-hub --port 8080 --dir /path/to/vega-missile
```

## API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/ask` | POST | Submit question (blocks until answered) |
| `/api/answer/{id}` | POST | Answer a pending question |
| `/api/questions` | GET | List pending questions |
| `/api/events` | GET | SSE stream for real-time updates |
| `/api/health` | GET | Health check |

## Hook Integration

The `PreToolUse` hook intercepts `AskUserQuestion` and routes to vega-hub:

```bash
# .claude/hooks/vega-hub-ask.sh
#!/bin/bash
# Extract question from stdin, POST to vega-hub, return answer
```

Response format:
```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": "[vega-hub] User answered: ..."
  }
}
```

## Project Structure

```
vega-hub/
├── cmd/vega-hub/       # Entry point
├── internal/
│   ├── api/            # HTTP handlers, SSE
│   ├── hub/            # Core state management
│   └── markdown/       # Goal file writing
├── web/                # React frontend
├── Dockerfile          # Production build
├── Dockerfile.dev      # Development with hot reload
└── docker-compose.yml  # Dev and build profiles
```

## License

MIT
