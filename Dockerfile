# Stage 1: Build the Go application binary (Go 1.25)
FROM golang:1.25-alpine AS builder

WORKDIR /app

ENV GOTOOLCHAIN=auto

# Install build tools if needed
RUN apk add --no-cache git

# Copy dependency manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build static CGO-free Go binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o kuspid_server ./cmd/server/main.go

# Stage 2: Final lightweight runtime container
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

# Copy compiled binary from builder stage
COPY --from=builder /app/kuspid_server /app/kuspid_server

# Copy web static assets and video segments
COPY web/ ./web/
COPY media/ ./media/

# Create data directory
RUN mkdir -p /app/data

EXPOSE 8080

ENV PORT=8080
ENV DB_PATH=/app/data/kuspid.db
ENV SEGMENTS_DIR=/app/media/segments

CMD ["/app/kuspid_server"]
