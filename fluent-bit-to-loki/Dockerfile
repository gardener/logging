#############      builder       #############
FROM golang:1.14.2 AS builder

WORKDIR /go/src/github.com/gardener/logging
COPY . .

RUN  if [ -d "fluent-bit-to-loki" ]; then cd fluent-bit-to-loki; fi; make plugin
#############      fluent-bit       #############
FROM fluent/fluent-bit:1.4.4 AS fluent-bit

COPY --from=builder /go/src/github.com/gardener/logging/fluent-bit-to-loki/build /fluent-bit/plugins

WORKDIR /

ENTRYPOINT ["/fluent-bit/bin/fluent-bit"]