#############      builder       #############
FROM golang:1.14.2 AS builder

WORKDIR /go/src/github.com/gardener/logging
COPY . .

RUN  make plugin
#############      fluent-bit       #############
FROM fluent/fluent-bit:1.5.4 AS fluent-bit

COPY --from=builder /go/src/github.com/gardener/logging/build /fluent-bit/plugins

WORKDIR /

ENTRYPOINT ["/fluent-bit/bin/fluent-bit"]