FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download || true

COPY cmd/ ./cmd/

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o certrenew ./cmd/certrenew

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata docker-cli curl bash

# Infisical CLI
RUN curl -1sLf 'https://artifacts-cli.infisical.com/setup.alpine.sh' | bash \
  && apk add --no-cache infisical

WORKDIR /app

COPY --from=builder /app/certrenew .
COPY infisical-run.sh .
RUN chmod +x infisical-run.sh

ENTRYPOINT ["./infisical-run.sh"]
