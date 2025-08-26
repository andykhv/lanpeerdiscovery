FROM golang:1.25 AS builder
WORKDIR /app

# Preload go.mod and go.sum first (better cache usage)
COPY go.mod ./
RUN go mod download

# Copy the entire project
COPY . .

# Build the discoveryd binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /discoveryd ./cmd/discoveryd

# ---- Runtime stage ----
FROM debian:bookworm-slim

# Small base image; install minimal tools (netcat, iproute2 are handy for debugging)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates iproute2 netcat-traditional && \
    rm -rf /var/lib/apt/lists/*

# Copy the binary from builder
COPY --from=builder /discoveryd /discoveryd

# Run as non-root user for safety
RUN useradd -m appuser
USER appuser

# Expose announce port (39999/udp) and echo port (40000/udp)
EXPOSE 39999/udp
EXPOSE 40000/udp

# Run the daemon
ENTRYPOINT ["/discoveryd"]
