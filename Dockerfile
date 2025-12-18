# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o dingdong -ldflags="-s -w" .

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create a non-root user
RUN adduser -D -g '' appuser

# Copy the binary from builder
COPY --from=builder /app/dingdong .

# Create data directory for Pocketbase
RUN mkdir -p /app/pb_data && chown -R appuser:appuser /app

USER appuser

# Expose the default Pocketbase port
EXPOSE 8090

# Volume for persistent data
VOLUME ["/app/pb_data"]

# Run the application
CMD ["./dingdong", "serve", "--http=0.0.0.0:8090"]
