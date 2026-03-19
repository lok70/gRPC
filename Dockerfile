FROM golang:1.21-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/grpc-auth ./cmd/sso && \
    CGO_ENABLED=0 GOOS=linux go build -o /out/migrator ./cmd/migrator

FROM alpine:3.20

RUN adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/grpc-auth /app/grpc-auth
COPY --from=builder /out/migrator /app/migrator
COPY config /app/config
COPY migrations /app/migrations

RUN mkdir -p /data && chown -R appuser:appuser /app /data

USER appuser

EXPOSE 44044

CMD ["sh", "-c", "./migrator --storage-path /data/sso.db --migrations-path ./migrations && ./grpc-auth --config ./config/docker.yaml"]
