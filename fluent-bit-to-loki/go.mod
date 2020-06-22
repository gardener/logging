module github.com/gardener/logging/fluent-bit-to-loki

go 1.13

require (
	github.com/ahmetb/gen-crd-api-reference-docs v0.2.0
	github.com/cortexproject/cortex v1.0.1-0.20200430170006-3462eb63f324
	github.com/fluent/fluent-bit-go v0.0.0-20190925192703-ea13c021720c
	github.com/gardener/gardener v1.5.0
	github.com/go-kit/kit v0.10.0
	github.com/go-logfmt/logfmt v0.5.0
	github.com/gobuffalo/packr/v2 v2.8.0
	github.com/golang/mock v1.4.3
	github.com/grafana/loki v1.4.1
	github.com/json-iterator/go v1.1.9
	github.com/onsi/ginkgo v1.10.3
	github.com/onsi/gomega v1.7.1
	github.com/prometheus/common v0.9.1
	github.com/weaveworks/common v0.0.0-20200429090833-ac38719f57dd
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.18.3
)

replace (
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v36.2.0+incompatible

	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.0+incompatible
	github.com/hpcloud/tail => github.com/grafana/tail v0.0.0-20191024143944-0b54ddf21fe7

	golang.org/x/net v0.0.0-20190813000000-74dc4d7220e7 => golang.org/x/net v0.0.0-20190813141303-74dc4d7220e7

	// Override reference that causes an error from Go proxy - see https://github.com/golang/go/issues/33558
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
)
