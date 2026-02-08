// Copyright 2026 Red Hat
// SPDX-License-Identifier: Apache-2.0

package vm_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVM(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VM Suite")
}
