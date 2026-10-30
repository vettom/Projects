# Stage 1: Build the Go application
FROM --platform=$BUILDPLATFORM golang:alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache ca-certificates git

# Cache Go modules layer
COPY go.mod go.sum ./
RUN go mod download

# Copy application source code
COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -trimpath -ldflags="-s -w" -o /app/promshim-http-sd .

# Stage 2: Create a minimal runtime image
FROM alpine:latest

# Install ca-certificates for Kubernetes API communication and create non-root user
RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/promshim-http-sd /app/promshim-http-sd

# Run as non-root user
USER 10001:10001

# Make port 9113 available outside this container
EXPOSE 9113

# Run the binary when container launches
ENTRYPOINT ["/app/promshim-http-sd"]
