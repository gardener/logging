# Notices for Fluent-bit-to-loki output plugin

This content is modification of [Grafana/Loki](https://github.com/grafana/loki) fluent-bit output plugin [v1.5.0](https://github.com/grafana/loki/tree/v1.5.0/cmd/fluent-bit).
  and is maintained by SAP.

## Copyright

All content is the property of the respective authors or their employers. For
more information regarding authorship of content, please consult the listed
source code repository logs.

## Modifications

After coping the original plugin from [Grafana/Loki](https://github.com/grafana/loki) the files, which was in one single direcotry, was splited to different packages. Then a controller package was added. The controller use shared informer to watch for namespaces and process the CREATE, UPDATE, DELETE Events. The `loki.go` file was modified to implement a function for storing the new controller and to search for the dynamic host path in the currently processiong log entry. The `config.go` was modified to parse additional properties needed for the new functionality. `loki.go` was split to `loki.go` and `utils.go` to separate methods from helper functions. `out_loki.go` was modified to initialize a incluster kubernetes client and make a shared informer which will be passed to each controller.