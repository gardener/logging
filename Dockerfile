#############      builder       #############
FROM golang:1.18 AS builder

WORKDIR /go/src/github.com/gardener/logging
COPY . .
ARG TARGETARCH
RUN make build GOARCH=$TARGETARCH

#############  fluent-bit-plugin #############
FROM alpine:3.15.4 AS fluent-bit-plugin

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
FROM telegraf:1.22.3 AS telegraf

RUN apt update
RUN apt install -y iptables
RUN apt clean

#############      eventlogger       #############
FROM gcr.io/distroless/static:nonroot AS event-logger

COPY --from=builder /go/src/github.com/gardener/logging/build/event-logger /event-logger

WORKDIR /

ENTRYPOINT [ "/event-logger" ]
