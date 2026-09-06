# --- STEP 1: Builder ---
FROM golang:1.26.4-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /restaurant-service ./cmd/restaurant-service

# --- STEP 2: Production ---
FROM alpine:3.24
WORKDIR /app
COPY --from=builder /restaurant-service .
CMD ["./restaurant-service"]
