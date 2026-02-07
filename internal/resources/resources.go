package resources

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EnsureNamespace creates a namespace with the given labels if it does not
// already exist. AlreadyExists errors are treated as success (idempotent).
func EnsureNamespace(ctx context.Context, c client.Client, name string, labels map[string]string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
	err := c.Create(ctx, ns)
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

// CreateService creates a Kubernetes Service. AlreadyExists errors are treated
// as success (idempotent).
func CreateService(ctx context.Context, c client.Client, svc *corev1.Service) error {
	err := c.Create(ctx, svc)
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

// DeleteManagedServices lists and deletes services matching the given labels in
// the namespace. Returns the count of successfully deleted services.
func DeleteManagedServices(ctx context.Context, c client.Client, namespace string, labels map[string]string) (int, error) {
	svcList := &corev1.ServiceList{}
	opts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(labels),
	}
	if err := c.List(ctx, svcList, opts...); err != nil {
		return 0, fmt.Errorf("listing services in %s: %w", namespace, err)
	}

	deleted := 0
	for i := range svcList.Items {
		if err := c.Delete(ctx, &svcList.Items[i]); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return deleted, fmt.Errorf("deleting service %s: %w", svcList.Items[i].Name, err)
		}
		deleted++
	}
	return deleted, nil
}
