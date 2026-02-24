# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install git for go modules
RUN apk add --no-cache git

# Copy go mod and sum files first for better caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o helix-agent ./cmd/agent

# Production stage
FROM alpine:latest AS production

RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# Copy binary from builder
COPY --from=builder /app/helix-agent /usr/local/bin/

# Create config directory
RUN mkdir -p /etc/helixops && \
    chown -R appuser:appgroup /etc/helixops

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the agent
ENTRYPOINT ["helix-agent"]
CMD ["--config", "/etc/helixops/config.yaml"]
