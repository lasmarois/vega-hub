# vega-hub

> Real-time human-executor communication hub for vega-missile.

## Project Structure

```
cmd/vega-hub/main.go     # Entry point with embedded web UI
internal/
├── api/                 # HTTP handlers and SSE
├── goals/               # Goal registry and file parsing
├── hub/                 # Core state, questions, executors, file watcher
└── markdown/            # Markdown file writing
web/                     # React + TypeScript frontend
```

## Quick Reference

| Task | Command |
|------|---------|
| Build binary | `sudo docker build -t vega-hub:dev -f Dockerfile .` |
| Run tests | `sudo docker run --rm -v "$(pwd)":/app -w /app golang:1.23-alpine go test ./internal/...` |
| Extract binary | See `rules/build.md` |

@rules/build.md
@rules/testing.md
