FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY . . 

RUN CGO_ENABLED=0 GOOS=linux go build -o gateway ./src/gateway/cmd/server

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/gateway .

EXPOSE 8080

CMD ["./gateway"]
