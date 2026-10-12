FROM golang:1.26.2-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/app \
    ./cmd/app

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app \
    && mkdir -p /var/log/food-delivery \
    && chown -R app:app /var/log/food-delivery

COPY --from=builder --chown=app:app /out/app /usr/local/bin/app

USER app

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/app"]
