# syntax=docker/dockerfile:1
FROM golang:alpine AS builder

WORKDIR /app

COPY . . 

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o gateway ./src/gateway/cmd/server

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/gateway .

EXPOSE 8080

CMD ["./gateway"]
