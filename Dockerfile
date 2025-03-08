#############      builder       #############
FROM golang:1.24.1 AS builder

WORKDIR /go/src/github.com/gardener/logging

COPY . .
RUN go mod download
RUN make plugin copy curator event-logger

############# distroless-static
FROM gcr.io/distroless/static-debian12:nonroot AS distroless-static

#############  fluent-bit-plugin #############
FROM distroless-static AS fluent-bit-plugin

COPY --from=builder /go/src/github.com/gardener/logging/build/out_vali.so /source/plugins/out_vali.so
COPY --from=builder /go/src/github.com/gardener/logging/build/copy /bin/cp

WORKDIR /

CMD ["/bin/cp", "/source/plugins/out_vali.so", "/plugins"]

#############  fluent-bit-vali #############
FROM ghcr.io/fluent/fluent-operator/fluent-bit:3.2.5 AS fluent-bit-vali

COPY --from=builder /go/src/github.com/gardener/logging/build/out_vali.so /fluent-bit/plugins/out_vali.so

WORKDIR /

CMD ["-e", "/fluent-bit/plugins/out_vali.so", "-c", "/fluent-bit/config/fluent-bit.conf"]

#############      curator       #############
FROM distroless-static AS curator

COPY --from=builder /go/src/github.com/gardener/logging/build/curator /curator

WORKDIR /
EXPOSE 2718

ENTRYPOINT [ "/curator" ]

#############      eventlogger       #############
FROM distroless-static AS event-logger

COPY --from=builder /go/src/github.com/gardener/logging/build/event-logger /event-logger

WORKDIR /

ENTRYPOINT [ "/event-logger" ]

#############      telegraf-builder       #############
FROM golang:1.24.1 AS telegraf-builder
RUN git clone --depth 1 --branch v1.26.0 https://github.com/influxdata/telegraf.git
WORKDIR /go/telegraf
ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target="/root/.cache/go-build" CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} make build

#############      iptables-builder       #############
FROM alpine:3.21.3 AS iptables-builder

RUN apk add --update bash sudo iptables ncurses-libs libmnl && \
    rm -rf /var/cache/apk/*

WORKDIR /volume

RUN mkdir -p ./bin ./sbin ./lib ./usr/bin ./usr/sbin ./usr/lib ./usr/lib/xtables ./usr/lib/bash ./tmp ./run ./etc/bash ./etc/openvpn ./usr/lib/openvpn/plugins ./etc/iproute2 ./etc/terminfo ./etc/logrotate.d ./etc/network/if-up.d ./usr/share/udhcpc ./etc/ssl/misc ./usr/lib/engines-1.1 ./run ./usr/lib/sudo \
    && cp -d /lib/ld-musl-* ./lib                                           && echo "package musl" \
    && cp -d /lib/libc.musl-* ./lib                                         && echo "package musl" \
    && cp -d -r /etc/terminfo/* ./etc/terminfo                              && echo "package ncurses-terminfo-base" \
    && cp -d /usr/lib/libformw.so.* ./usr/lib                               && echo "package ncurses-libs" \
    && cp -d /usr/lib/libmenuw.so.* ./usr/lib                               && echo "package ncurses-libs" \
    && cp -d /usr/lib/libncursesw.so.* ./usr/lib                            && echo "package ncurses-libs" \
    && cp -d /usr/lib/libpanelw.so.* ./usr/lib                              && echo "package ncurses-libs" \
    && cp -d /usr/lib/libreadline.so.* ./usr/lib                            && echo "package readline" \
    && cp -d /etc/inputrc ./etc                                             && echo "package readline" \
    && cp -d /bin/bash ./bin                                                && echo "package bash" \
    && cp -d /etc/bash/bashrc ./etc/bash                                    && echo "package bash" \
    && cp -d /usr/lib/bash/* ./usr/lib/bash                                 && echo "package bash" \
    && cp -d /usr/lib/libz.* ./lib                                          && echo "package zlib" \
    && cp -d /usr/lib/libmnl.* ./usr/lib                                    && echo "package libmnl" \
    && cp -d /usr/lib/libnftnl* ./usr/lib                                   && echo "package libnftnl" \
    && cp -d /etc/ethertypes ./etc                                          && echo "package iptables" \
    && cp -d /usr/sbin/iptables* ./sbin                                     && echo "package iptables" \
    && cp -d /usr/sbin/xtables* ./sbin                                      && echo "package iptables" \
    && cp -d /usr/lib/libxtables* ./usr/lib                                 && echo "package iptables" \
    && cp -d /usr/lib/xtables/* ./usr/lib/xtables                           && echo "package iptables" \
    && cp -d /usr/lib/sudo/* ./usr/lib/sudo                                 && echo "package sudo" \
    && cp -d /etc/sudoers ./etc                                             && echo "package sudo" \
    && cp -d /etc/passwd ./etc                                              && echo "package sudo" \
    && cp -d /usr/bin/sudo ./usr/sbin                                       && echo "package sudo" \
    && touch ./run/xtables.lock                                             && echo "create /run/xtables.lock"

#############      telegraf       #############
FROM scratch AS telegraf

COPY --from=iptables-builder /volume /

COPY --from=telegraf-builder /go/telegraf/telegraf /usr/bin/telegraf

CMD [ "/usr/bin/telegraf"]

#############      tune2fs-builder       #############
FROM alpine:3.21.3 AS tune2fs-builder

RUN apk add --update bash e2fsprogs-extra mount gawk ncurses-libs && \
    rm -rf /var/cache/apk/*

WORKDIR /volume

RUN mkdir -p ./lib ./usr/bin/ ./bin ./etc/bash ./usr/lib/bash ./usr/sbin/ ./etc/terminfo \
    && cp -d /usr/bin/gawk ./usr/bin                                        && echo "package gawk" \
    && cp -d /lib/ld-musl-* ./lib                                           && echo "package musl" \
    && cp -d /lib/libc.musl-* ./lib                                         && echo "package musl" \
    && cp -d /usr/lib/libmount.so.* ./lib                                   && echo "package libmount" \
    && cp -d /usr/lib/libblkid.so.* ./lib                                   && echo "package libblkid" \
    && cp -d /bin/mount ./bin                                               && echo "package mount" \
    && cp -d -r /etc/terminfo/* ./etc/terminfo                              && echo "package ncurses-terminfo-base" \
    && cp -d /usr/lib/libformw.so.* ./usr/lib                               && echo "package ncurses-libs" \
    && cp -d /usr/lib/libmenuw.so.* ./usr/lib                               && echo "package ncurses-libs" \
    && cp -d /usr/lib/libncursesw.so.* ./usr/lib                            && echo "package ncurses-libs" \
    && cp -d /usr/lib/libpanelw.so.* ./usr/lib                              && echo "package ncurses-libs" \
    && cp -d /usr/lib/libreadline.so.* ./usr/lib                            && echo "package readline" \
    && cp -d /etc/inputrc ./etc                                             && echo "package readline" \
    && cp -d /bin/bash ./bin                                                && echo "package bash" \
    && cp -d /etc/bash/bashrc ./etc/bash                                    && echo "package bash" \
    && cp -d /usr/lib/bash/* ./usr/lib/bash                                 && echo "package bash" \
    && cp -d /usr/lib/libext2fs.so.* ./lib                                  && echo "package e2fsprogs-extra" \
    && cp -d /usr/lib/libcom_err.so.* ./lib                                 && echo "package e2fsprogs-extra" \
    && cp -d /usr/lib/libuuid.so.* ./lib                                    && echo "package e2fsprogs-extra" \
    && cp -d /usr/lib/libe2p.so.* ./lib                                     && echo "package e2fsprogs-extra" \
    && cp -d /usr/lib/libeconf.so.* ./lib                                   && echo "package libeconf" \
    && cp -d /usr/sbin/tune2fs ./usr/sbin                                   && echo "package e2fsprogs-extra"

#############      tune2fs       #############
FROM scratch AS tune2fs

COPY --from=tune2fs-builder /volume /

CMD [ "/usr/sbin/tune2fs"]
