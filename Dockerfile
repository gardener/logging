#############      builder       #############
FROM golang:1.19.1 AS plugin-builder

WORKDIR /go/src/github.com/gardener/logging
COPY . .

ARG TARGETARCH
RUN make plugin GOARCH=$TARGETARCH
RUN make install-copy

############# distroless-static
FROM gcr.io/distroless/static-debian11:nonroot as distroless-static

#############  fluent-bit-plugin #############
FROM distroless-static AS fluent-bit-plugin

COPY --from=plugin-builder /go/src/github.com/gardener/logging/build /source/plugins
COPY --from=plugin-builder /go/bin/copy /bin/cp

WORKDIR /

CMD /bin/cp /source/plugins/. /plugins

#############      image-builder       #############
FROM golang:1.19.1 AS image-builder

WORKDIR /go/src/github.com/gardener/logging
COPY . .

ARG EFFECTIVE_VERSION
ARG TARGETARCH
RUN make install EFFECTIVE_VERSION=$EFFECTIVE_VERSION GOARCH=$TARGETARCH

#############      curator       #############
FROM distroless-static AS curator

COPY --from=image-builder /go/bin/loki-curator /curator

WORKDIR /
EXPOSE 2718

ENTRYPOINT [ "/curator" ]

#############      eventlogger       #############
FROM distroless-static AS event-logger

COPY --from=image-builder /go/bin/event-logger /event-logger

WORKDIR /

ENTRYPOINT [ "/event-logger" ]

#############      telegraf       #############
FROM telegraf:1.23.4 AS telegraf

RUN apt update
RUN apt install -y iptables
RUN apt clean