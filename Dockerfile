#############      builder       #############
FROM golang:1.14.2 AS builder

WORKDIR /go/src/github.com/gardener/logging
COPY . .

RUN  make plugin
#############      carrier       #############
FROM alpine:3.12.0 AS carrier

COPY --from=builder /go/src/github.com/gardener/logging/build /source/plugins

WORKDIR /
