#############      builder       #############
FROM golang:1.18 AS builder

WORKDIR /go/src/github.com/gardener/logging
COPY . .

RUN  make build

#############  fluent-bit-plugin #############
FROM eu.gcr.io/gardener-project/3rd/alpine:3.12.3 AS fluent-bit-plugin

COPY --from=builder /go/src/github.com/gardener/logging/build /source/plugins

WORKDIR /

ENTRYPOINT ["cp","/source/plugins/.","/plugins", "-fr"]

#############      curator       #############
FROM gcr.io/distroless/static:nonroot AS curator

COPY --from=builder /go/src/github.com/gardener/logging/build/curator /curator

WORKDIR /
EXPOSE 2718

ENTRYPOINT [ "/curator" ]

#############      telegraf       #############
FROM telegraf:1.18.0-alpine AS telegraf

RUN apk add --update bash iptables su-exec sudo && rm -rf /var/cache/apk/*
