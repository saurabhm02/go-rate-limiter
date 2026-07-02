# Production image: static server + migrate binary, non-root runtime.
FROM golang:1.25-alpine AS build

WORKDIR /src
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/server ./cmd/server \
    && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM alpine:3.19

RUN apk add --no-cache ca-certificates wget \
    && adduser -D -H -u 65532 app

COPY --from=build /out/server /server
COPY --from=build /out/migrate /migrate

USER 65532
EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/health/live || exit 1

ENTRYPOINT ["/server"]
