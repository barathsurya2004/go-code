FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY . . 

RUN CGO_ENABLED=0 GOOS=linux go build -o penne-service ./src/penne-service/cmd/server

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/penne-service .

EXPOSE 8080

CMD ["./penne-service"]

