#############      builder       #############
FROM golang:1.18.5 AS plugin-builder

WORKDIR /go/src/github.com/gardener/logging
COPY . .

ARG TARGETARCH
RUN make plugin GOARCH=$TARGETARCH

#############  fluent-bit-plugin #############
FROM alpine:3.16.2 AS fluent-bit-plugin

COPY --from=plugin-builder /go/src/github.com/gardener/logging/build /source/plugins

WORKDIR /

CMD cp /source/plugins/. /plugins -fr

#############      image-builder       #############
FROM golang:1.18.5 AS image-builder

WORKDIR /go/src/github.com/gardener/logging
COPY . .

ARG EFFECTIVE_VERSION
ARG TARGETARCH
RUN make install EFFECTIVE_VERSION=$EFFECTIVE_VERSION GOARCH=$TARGETARCH

#############      curator       #############
FROM gcr.io/distroless/static:nonroot AS curator

COPY --from=image-builder /go/bin/loki-curator /curator

WORKDIR /
EXPOSE 2718

ENTRYPOINT [ "/curator" ]

#############      eventlogger       #############
FROM gcr.io/distroless/static:nonroot AS event-logger

COPY --from=image-builder /go/bin/event-logger /event-logger

WORKDIR /

ENTRYPOINT [ "/event-logger" ]

#############      telegraf       #############
FROM telegraf:1.23.4 AS telegraf

RUN apt update
RUN apt install -y iptables
RUN apt clean