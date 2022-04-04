package tinysl_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTinysl(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tinysl Suite")
}
