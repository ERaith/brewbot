### Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=dev
RUN go build -ldflags "-X main.Version=${VERSION}" -o brewbot .

### Runtime stage
FROM alpine:latest

RUN adduser -D -u 1001 brewbot
WORKDIR /app

COPY --from=builder /app/brewbot .

USER brewbot
ENTRYPOINT ["./brewbot"]
