# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server
RUN CGO_ENABLED=0 go build -o plumber-server cmd/plumber-server/main.go

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/plumber-server .

# Create config directory
RUN mkdir -p /app/configs

# Expose the server port
EXPOSE 52281

# Run the server with config file
CMD ["./plumber-server"]
