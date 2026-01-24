# Build stage for React frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

# Copy frontend package files
COPY web/package*.json ./
RUN npm install

# Copy frontend source and build
COPY web/ ./
RUN npm run build

# Build stage for Go binary
FROM golang:1.23-alpine AS backend-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend into embed location (inside cmd/vega-hub for embed)
COPY --from=frontend-builder /app/web/dist ./cmd/vega-hub/web

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o vega-hub ./cmd/vega-hub

# Final minimal image
FROM alpine:3.19

WORKDIR /app

# Copy the binary
COPY --from=backend-builder /app/vega-hub .

# Expose default port
EXPOSE 8080

ENTRYPOINT ["./vega-hub"]
