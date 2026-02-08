// Copyright 2026 Red Hat
// SPDX-License-Identifier: Apache-2.0

package cleanup_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCleanup(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cleanup Suite")
}
