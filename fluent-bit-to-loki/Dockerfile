#############      builder       #############
FROM golang:1.14.2 AS builder

WORKDIR /go/src/github.com/gardener/logging/fluent-bit-to-loki
COPY ./fluent-bit-to-loki .

RUN  make plugin
#############      fluent-bit       #############
FROM fluent/fluent-bit:1.5.4 AS fluent-bit

COPY --from=builder /go/src/github.com/gardener/logging/fluent-bit-to-loki/build /fluent-bit/plugins

WORKDIR /

ENTRYPOINT ["/fluent-bit/bin/fluent-bit"]