//go:build integration

// Copyright 2026 Red Hat
// SPDX-License-Identifier: Apache-2.0

package cluster_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kubevirtv1 "kubevirt.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"virtwork/internal/cluster"
)

var _ = Describe("Connect [integration]", func() {
	It("should connect via default kubeconfig resolution", func() {
		c, err := cluster.Connect("")
		Expect(err).NotTo(HaveOccurred())
		Expect(c).NotTo(BeNil())
	})

	It("should return a functional client that can list namespaces", func() {
		c, err := cluster.Connect("")
		Expect(err).NotTo(HaveOccurred())

		nsList := &corev1.NamespaceList{}
		err = c.List(context.Background(), nsList)
		Expect(err).NotTo(HaveOccurred())
		Expect(nsList.Items).NotTo(BeEmpty())
	})

	It("should register KubeVirt types", func() {
		c, err := cluster.Connect("")
		Expect(err).NotTo(HaveOccurred())

		// Listing VMs should not return a "no kind registered" error.
		// An empty list is fine — we just need the type to be recognized.
		vmList := &kubevirtv1.VirtualMachineList{}
		err = c.List(context.Background(), vmList, client.InNamespace("default"))
		Expect(err).NotTo(HaveOccurred())
	})

	It("should return error for invalid kubeconfig path", func() {
		_, err := cluster.Connect("/nonexistent/kubeconfig")
		Expect(err).To(HaveOccurred())
	})
})
