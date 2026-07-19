FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/navx-server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/navx-server ./navx-server

# Render sets $PORT and injects it here; main.go already reads it (falls back
# to 8080 for any other environment that doesn't set it).
EXPOSE 8080
CMD ["./navx-server"]
