// Copyright 2026 Red Hat
// SPDX-License-Identifier: Apache-2.0

package cleanup

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubevirtv1 "kubevirt.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"virtwork/internal/constants"
)

// CleanupResult summarises the outcome of a cleanup operation.
type CleanupResult struct {
	VMsDeleted       int
	ServicesDeleted  int
	SecretsDeleted   int
	NamespaceDeleted bool
	Errors           []error
}

// CleanupAll deletes all virtwork-managed resources in the given namespace.
// Individual deletion failures are recorded but do not abort the operation.
// If deleteNamespace is true, the namespace itself is deleted as the final step.
func CleanupAll(ctx context.Context, c client.Client, namespace string, deleteNamespace bool) (*CleanupResult, error) {
	result := &CleanupResult{}
	managedLabels := map[string]string{
		constants.LabelManagedBy: constants.ManagedByValue,
	}

	// Delete VMs by label
	vmList := &kubevirtv1.VirtualMachineList{}
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(managedLabels),
	}
	if err := c.List(ctx, vmList, listOpts...); err != nil {
		return result, fmt.Errorf("listing VMs in %s: %w", namespace, err)
	}
	for i := range vmList.Items {
		if err := c.Delete(ctx, &vmList.Items[i]); err != nil {
			if !apierrors.IsNotFound(err) {
				result.Errors = append(result.Errors, fmt.Errorf("deleting VM %s: %w", vmList.Items[i].Name, err))
			}
			continue
		}
		result.VMsDeleted++
	}

	// Delete services by label
	svcList := &corev1.ServiceList{}
	if err := c.List(ctx, svcList, listOpts...); err != nil {
		return result, fmt.Errorf("listing services in %s: %w", namespace, err)
	}
	for i := range svcList.Items {
		if err := c.Delete(ctx, &svcList.Items[i]); err != nil {
			if !apierrors.IsNotFound(err) {
				result.Errors = append(result.Errors, fmt.Errorf("deleting service %s: %w", svcList.Items[i].Name, err))
			}
			continue
		}
		result.ServicesDeleted++
	}

	// Delete secrets by label
	secretList := &corev1.SecretList{}
	if err := c.List(ctx, secretList, listOpts...); err != nil {
		return result, fmt.Errorf("listing secrets in %s: %w", namespace, err)
	}
	for i := range secretList.Items {
		if err := c.Delete(ctx, &secretList.Items[i]); err != nil {
			if !apierrors.IsNotFound(err) {
				result.Errors = append(result.Errors, fmt.Errorf("deleting secret %s: %w", secretList.Items[i].Name, err))
			}
			continue
		}
		result.SecretsDeleted++
	}

	// Optionally delete namespace
	if deleteNamespace {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		if err := c.Delete(ctx, ns); err != nil {
			if !apierrors.IsNotFound(err) {
				result.Errors = append(result.Errors, fmt.Errorf("deleting namespace %s: %w", namespace, err))
			}
		} else {
			result.NamespaceDeleted = true
		}
	}

	return result, nil
}
