FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY . . 

RUN CGO_ENABLED=0 GOOS=linux go build -o penne-service ./src/penne-service/cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o cadence-worker ./src/penne-service/cmd/cadence

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/penne-service .
COPY --from=builder /app/cadence-worker .

EXPOSE 8080 50051

CMD ["./penne-service"]
