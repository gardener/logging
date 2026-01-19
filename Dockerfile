#############      builder       #############
FROM golang:1.26rc2 AS builder

WORKDIR /go/src/github.com/gardener/logging

COPY . .
RUN go mod download
RUN make plugin copy event-logger

############# distroless-static
FROM gcr.io/distroless/static-debian12:nonroot AS distroless-static

#############  fluent-bit-plugin #############
FROM distroless-static AS fluent-bit-plugin

COPY --from=builder /go/src/github.com/gardener/logging/build/output_plugin.so /source/plugins/output_plugin.so
COPY --from=builder /go/src/github.com/gardener/logging/build/copy /bin/cp

WORKDIR /

CMD ["/bin/cp", "/source/plugins/output_plugin.so", "/plugins"]

#############  fluent-bit-output #############
FROM ghcr.io/fluent/fluent-operator/fluent-bit:4.2.0 AS fluent-bit-output

COPY --from=builder /go/src/github.com/gardener/logging/build/output_plugin.so /fluent-bit/plugins/output_plugin.so

WORKDIR /

CMD ["-e", "/fluent-bit/plugins/output_plugin.so", "-c", "/fluent-bit/config/fluent-bit.conf"]

#############      eventlogger       #############
FROM distroless-static AS event-logger

COPY --from=builder /go/src/github.com/gardener/logging/build/event-logger /event-logger

WORKDIR /

ENTRYPOINT [ "/event-logger" ]

#############      tune2fs-builder       #############
FROM alpine:3.23.2 AS tune2fs-builder

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
