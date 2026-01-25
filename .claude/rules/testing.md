# Testing vega-hub

> Docker-based test execution for Go packages.

## Test Commands

**Run all tests:**
```bash
sudo docker run --rm -v "$(pwd)":/app -w /app golang:1.23-alpine go test ./internal/...
```

**Run tests with verbose output:**
```bash
sudo docker run --rm -v "$(pwd)":/app -w /app golang:1.23-alpine go test -v ./internal/...
```

**Run specific package tests:**
```bash
sudo docker run --rm -v "$(pwd)":/app -w /app golang:1.23-alpine go test ./internal/hub/...
sudo docker run --rm -v "$(pwd)":/app -w /app golang:1.23-alpine go test ./internal/goals/...
sudo docker run --rm -v "$(pwd)":/app -w /app golang:1.23-alpine go test ./internal/api/...
```

**Run single test:**
```bash
sudo docker run --rm -v "$(pwd)":/app -w /app golang:1.23-alpine go test -v -run TestName ./internal/hub/...
```

## DO

- Use `sudo docker run` with `golang:1.23-alpine` image
- Test `./internal/...` (excludes cmd/ which needs embedded files)
- Mount current directory with `-v "$(pwd)":/app`
- Run from project root directory

## DO NOT

- Run `go test` directly (Go not installed on host)
- Test `./...` (fails on cmd/ due to embed directive)
- Run `make test` (uses Docker, may work but verbose)
- Skip tests after code changes

## Test Coverage

| Package | Coverage |
|---------|----------|
| `internal/goals` | Registry parsing, goal detail parsing |
| `internal/hub` | Executor lifecycle, Q&A flow, SSE, file watcher |
| `internal/api` | HTTP handlers, CORS, routing |
| `internal/markdown` | No tests (simple writer) |

## Writing New Tests

1. Create `*_test.go` in same package
2. Use `t.TempDir()` for test directories
3. Use `setupTestEnv(t)` helpers when available
4. Test both success and error cases
