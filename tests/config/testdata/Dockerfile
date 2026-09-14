FROM alpine:3 AS certs
RUN apk add --no-cache openssl && \
    openssl genrsa -out /ca.key 2048 && \
    openssl req -x509 -new -noenc -key /ca.key -subj "/CN=test-ca" -days 10000 -out /ca.crt && \
    openssl genrsa -out /test.key 2048 && \
    openssl req -new -key /test.key -subj "/CN=test-server" -out /test.csr && \
    openssl x509 -req -in /test.csr -CA /ca.crt -CAkey /ca.key -CAcreateserial -days 10000 -sha256 -out /test.crt

FROM ghcr.io/fluent/fluent-operator/fluent-bit:v5.1.1
COPY --from=certs /test.crt /etc/ssl/certs/test.crt
COPY --from=certs /test.key /etc/ssl/private/test.key
COPY --from=certs /test.crt /etc/ssl/certs/ca.crt
COPY output_plugin.so /fluent-bit/plugins/output_plugin.so
COPY config /fluent-bit/config/fluent-bit.{{ .ConfigExt }}
ENTRYPOINT ["/fluent-bit/bin/fluent-bit", "-c", "/fluent-bit/config/fluent-bit.{{ .ConfigExt }}", "-e", "/fluent-bit/plugins/output_plugin.so"]
