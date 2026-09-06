# --- STEP 1: Builder ---
FROM golang:1.26.4-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server cmd/server/main.go

# --- STEP 2: Production ---
FROM alpine:3.24
WORKDIR /app
COPY --from=builder /app/server .

CMD ["./server"]
