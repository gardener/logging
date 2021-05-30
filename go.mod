module github.com/gardener/logging

go 1.14

require (
	github.com/ahmetb/gen-crd-api-reference-docs v0.2.0
	github.com/cortexproject/cortex v1.2.1-0.20200803161316-7014ff11ed70
	github.com/fluent/fluent-bit-go v0.0.0-20190925192703-ea13c021720c
	github.com/gardener/gardener v1.5.0
	github.com/go-kit/kit v0.10.0
	github.com/go-logfmt/logfmt v0.5.0
	github.com/gobuffalo/packr/v2 v2.8.0
	github.com/golang/mock v1.4.3
	github.com/google/go-cmp v0.5.1 // indirect
	github.com/grafana/loki v1.6.0
	github.com/joncrlsn/dque v2.2.1-0.20200515025108-956d14155fa2+incompatible
	github.com/json-iterator/go v1.1.10
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.8.0
	github.com/prometheus/common v0.14.0
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/cobra v1.1.3 // indirect
	github.com/weaveworks/common v0.0.0-20200625145055-4b1847531bc9
	golang.org/x/sys v0.0.0-20210403161142-5e06dd20ab57 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.18.3
	k8s.io/utils v0.0.0-20200414100711-2df71ebbae66
)

replace (
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v36.2.0+incompatible

	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.0+incompatible
	github.com/hpcloud/tail => github.com/grafana/tail v0.0.0-20191024143944-0b54ddf21fe7
	github.com/prometheus/common => github.com/prometheus/common v0.10.0

	golang.org/x/net v0.0.0-20190813000000-74dc4d7220e7 => golang.org/x/net v0.0.0-20190813141303-74dc4d7220e7
	k8s.io/api => k8s.io/api v0.17.9
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.9
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.9
	k8s.io/apiserver => k8s.io/apiserver v0.17.9
	k8s.io/client-go => k8s.io/client-go v0.17.9
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.9
	k8s.io/code-generator => k8s.io/code-generator v0.17.9
	k8s.io/component-base => k8s.io/component-base v0.17.9
	k8s.io/helm => k8s.io/helm v2.13.1+incompatible
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.9
// Override reference that causes an error from Go proxy - see https://github.com/golang/go/issues/33558
//k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
)
