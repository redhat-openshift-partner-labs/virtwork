// Copyright 2026 Red Hat
// SPDX-License-Identifier: Apache-2.0

package constants_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConstants(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Constants Suite")
}
