FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download || true

COPY cmd/ ./cmd/

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o certrenew ./cmd/certrenew

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata docker-cli

WORKDIR /app

COPY --from=builder /app/certrenew .

ENTRYPOINT ["./certrenew"]
