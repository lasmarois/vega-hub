.PHONY: dev build clean test frontend-init

# Development: run frontend and backend with hot reload
dev:
	docker compose --profile dev up

# Build production binary
build:
	docker compose --profile build up --build
	@echo "Binary available at: ./dist/vega-hub"

# Initialize frontend (first time only)
frontend-init:
	docker run --rm -v $(PWD)/web:/app -w /app node:20-alpine sh -c "npm create vite@latest . -- --template react-ts && npm install"

# Add shadcn to frontend
frontend-shadcn:
	docker run --rm -v $(PWD)/web:/app -w /app node:20-alpine sh -c "npx shadcn@latest init -y && npx shadcn@latest add button card input"

# Clean build artifacts
clean:
	rm -rf dist/
	docker compose down -v

# Run tests
test:
	docker run --rm -v $(PWD):/app -w /app golang:1.23-alpine go test ./...

# Build and run locally (for quick testing)
run: build
	./dist/vega-hub --port 8080
