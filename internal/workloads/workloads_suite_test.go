// Copyright 2026 Red Hat
// SPDX-License-Identifier: Apache-2.0

package workloads_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestWorkloads(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Workloads Suite")
}
