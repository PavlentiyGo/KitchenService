FROM golang:1.26.2-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/mock-service \
    ./cmd/mock-service

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app

COPY --from=builder --chown=app:app /out/mock-service /usr/local/bin/mock-service

USER app

EXPOSE 8081

ENTRYPOINT ["/usr/local/bin/mock-service"]
