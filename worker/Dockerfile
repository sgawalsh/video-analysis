# Stage 1: Build
FROM golang:1-alpine3@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS builder
RUN apk add --no-cache git

WORKDIR /src

# Copy go.mod and go.sum for the worker module
COPY src/go.mod .
COPY src/go.sum .

# Download dependencies
RUN go mod download

# Copy worker source
COPY src/ .

# Build binary
RUN go build -o worker .

# Stage 2: Minimal image
FROM alpine:3@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf

# Install runtime dependencies
RUN apk upgrade --no-cache && apk add --no-cache \
    python3 \
    py3-pip \
    ffmpeg \
    && pip install --break-system-packages yt-dlp

RUN adduser -D worker

COPY --from=builder /src/worker /worker

USER worker
WORKDIR /

CMD ["/worker"]