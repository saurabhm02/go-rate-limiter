# Build example service binary (SERVICE=payment-service|order-service|gateway-middleware)
FROM golang:1.22-alpine AS build

ARG SERVICE
WORKDIR /src
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/service ./examples/${SERVICE}

FROM alpine:3.19

RUN apk add --no-cache ca-certificates wget \
    && adduser -D -H -u 65532 app

COPY --from=build /out/service /service

USER 65532

ENTRYPOINT ["/service"]
