FROM ghcr.io/fluent/fluent-operator/fluent-bit:v5.1.1
COPY output_plugin.so /fluent-bit/plugins/output_plugin.so
COPY config /fluent-bit/config/fluent-bit.{{ .ConfigExt }}
ENTRYPOINT ["/fluent-bit/bin/fluent-bit", "-c", "/fluent-bit/config/fluent-bit.{{ .ConfigExt }}", "-e", "/fluent-bit/plugins/output_plugin.so"]
