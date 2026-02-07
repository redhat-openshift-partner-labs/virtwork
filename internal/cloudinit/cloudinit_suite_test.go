package cloudinit_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCloudinit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cloudinit Suite")
}
