FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /idrac6-manager ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /idrac6-manager /usr/local/bin/idrac6-manager

EXPOSE 8080

ENTRYPOINT ["idrac6-manager"]
