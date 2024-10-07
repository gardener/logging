package vali_plugin_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOutputPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Test")
}
