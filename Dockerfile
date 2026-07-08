# ============ Build Stage ============
FROM golang:1.23-alpine AS builder

# CGO_ENABLED=0 不需要 gcc/musl-dev
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
	-ldflags="-s -w" \
	-o /app/server ./cmd/server/

# ============ Runtime Stage ============
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/.env.example .env

EXPOSE 8080

ENTRYPOINT ["./server"]