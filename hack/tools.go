// +build tools

// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// This package imports things required by build scripts, to force `go mod` to see them as dependencies
package tools

import (
	_ "github.com/gardener/gardener/hack"
	_ "github.com/gardener/gardener/hack/.ci"
	_ "github.com/gardener/gardener/hack/api-reference/template"

	_ "github.com/ahmetb/gen-crd-api-reference-docs"
	_ "github.com/gobuffalo/packr/v2/packr2"
	_ "github.com/golang/mock/mockgen"
	_ "github.com/onsi/ginkgo/ginkgo"
	_ "k8s.io/code-generator"
)
