package tests

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestKsp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "intigration")
}

var _ = Describe("intigration", func() {

	Describe("Match syscalls", func() {
		It("can detect unlink syscall", func() {})
	})

})
