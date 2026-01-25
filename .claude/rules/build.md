# Building vega-hub

> Docker-based build system for cross-platform binary.

## Build Commands

**Build the binary:**
```bash
sudo docker build -t vega-hub:dev -f Dockerfile .
```

**Extract the binary:**
```bash
sudo docker create --name vega-hub-extract vega-hub:dev
sudo docker cp vega-hub-extract:/app/vega-hub ./vega-hub
sudo docker rm vega-hub-extract
```

**Deploy to vega-missile:**
```bash
sudo docker cp vega-hub-extract:/app/vega-hub /path/to/vega-missile/tools/vega-hub
sudo chown $USER:$GROUP /path/to/vega-missile/tools/vega-hub
```

## DO

- Use `sudo docker build` (Docker requires elevated permissions)
- Use `Dockerfile` at project root (multi-stage build)
- Extract binary after build (binary is inside container)
- Change ownership after extraction (Docker creates as root)

## DO NOT

- Run `go build` directly (Go not installed on host)
- Run `make build` (uses docker-compose, may fail)
- Skip the extraction step (binary stays in container)
- Run the binary as root (won't find claude CLI)

## What the Build Does

1. **Frontend stage**: `node:20-alpine` builds React app with Vite
2. **Backend stage**: `golang:1.23-alpine` compiles Go with embedded frontend
3. **Final stage**: `alpine:3.19` minimal image with just the binary

## Common Build Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `permission denied` | Docker socket access | Use `sudo` |
| `pattern all:web: no matching files` | Missing frontend build | Let Dockerfile handle it |
| TypeScript errors | Frontend code issues | Fix in `web/src/` |
