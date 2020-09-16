# Notices for Fluent-bit-to-loki output plugin

This plugin extends [Grafana/Loki](https://github.com/grafana/loki) [fluent-bit output plugin v1.6.0](https://github.com/grafana/loki/tree/v1.6.0/cmd/fluent-bit) which aims to forward log messages from fluent-bit to Loki. It is maintained by SAP.

## Copyright

All content is the property of the respective authors or their employers. For
more information regarding authorship of content, please consult the listed
source code repository logs.

## Modifications

After coping the original plugin from [Grafana/Loki](https://github.com/grafana/loki/tree/v1.6.0/cmd/fluent-bit) the files, which was in one single directory, was splitted to different packages. Then a controller package was added. The controller use shared informer to watch for namespaces and process the CREATE, UPDATE, DELETE Events. The `loki.go` file was modified to implement a function for storing the new controller and to search for the dynamic host path in the currently processing log entry. The `config.go` was modified to parse additional properties needed for the new functionality. `loki.go` was split to `loki.go` and `utils.go` to separate methods from helper functions. `out_loki.go` was modified to initialize cluster kubernetes client and make a shared informer which will be passed to each controller.