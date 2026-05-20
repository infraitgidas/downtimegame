# =============================================================================
# Backend — Downtime Game
# Multi-stage: build Go binary, then run in minimal image
# =============================================================================

# --- Builder ---
FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

# --- Runtime ---
FROM alpine:3.21

RUN apk add --no-cache ca-certificates
COPY --from=builder /server /server

EXPOSE 8080
CMD ["/server"]
