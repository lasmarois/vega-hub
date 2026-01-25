# vega-hub

[![Release](https://img.shields.io/github/v/release/lasmarois/vega-hub)](https://github.com/lasmarois/vega-hub/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Real-time communication hub for human-executor interaction in [vega-missile](https://github.com/lasmarois/vega-missile).

## Overview

vega-hub enables direct communication between humans and Claude Code executors via:
- Hook interception of `AskUserQuestion` tool calls
- Web UI for answering questions in real-time
- Markdown persistence for Q&A history
- Executor lifecycle management (spawn, monitor, stop)
- Desktop notifications when executors need attention

## Installation

### With vega-missile (Recommended)

vega-hub is automatically downloaded when you start a vega-missile session. The version is pinned in `tools/.vega-hub-version`.

### Standalone

Download the latest release for your platform:

```bash
# Linux (amd64)
curl -sL https://github.com/lasmarois/vega-hub/releases/latest/download/vega-hub-linux-amd64 -o vega-hub
chmod +x vega-hub

# macOS (arm64)
curl -sL https://github.com/lasmarois/vega-hub/releases/latest/download/vega-hub-darwin-arm64 -o vega-hub
chmod +x vega-hub
```

Available binaries: `linux-amd64`, `linux-arm64`, `darwin-amd64`, `darwin-arm64`

### Run

```bash
./vega-hub --port 8080 --dir /path/to/vega-missile
```

## Development

### Build from Source

```bash
make build
```

This creates `./dist/vega-hub` - a single binary with embedded React UI.

### Development Mode

```bash
# Start frontend (hot reload) and backend (hot reload)
make dev
```

- Frontend: http://localhost:5173
- Backend API: http://localhost:8080

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

## vega-missile Integration

vega-hub is designed to work with [vega-missile](https://github.com/lasmarois/vega-missile), a multi-project orchestration system for Claude Code.

```
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│  Manager (you)   │────▶│    vega-hub      │◀────│    Executors     │
│  Claude session  │     │  localhost:8080  │     │  (in worktrees)  │
└──────────────────┘     └──────────────────┘     └──────────────────┘
```

**How it works:**
1. Manager starts vega-hub (automatic on session start)
2. Manager spawns executors via vega-hub API
3. Executors route `AskUserQuestion` calls through vega-hub hooks
4. Human answers questions in the web UI
5. vega-hub persists Q&A to goal markdown files

## Releases

Releases are automated via GitHub Actions. To create a new release:

1. Update `VERSION` file with new version (e.g., `0.3.0`)
2. Update `CHANGELOG.md` with release notes
3. Push to master
4. GitHub Actions builds binaries and creates the release

## License

MIT
