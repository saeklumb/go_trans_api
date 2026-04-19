FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN go build -o /bin/tx-service ./cmd/app

FROM alpine:3.22
WORKDIR /app
COPY --from=builder /bin/tx-service /usr/local/bin/tx-service

EXPOSE 8080
CMD ["tx-service"]

