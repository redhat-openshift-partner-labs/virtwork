//go:build e2e

// Copyright 2026 Red Hat
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opdev/virtwork/internal/testutil"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func() {
	// Ensure the binary is built before any test runs.
	// BinaryPath() builds on first call and caches the result.
	path, err := testutil.BinaryPath()
	Expect(err).NotTo(HaveOccurred(), "Failed to build virtwork binary")
	GinkgoWriter.Printf("Using virtwork binary: %s\n", path)
})
