package service

import (
	"context"
	"fmt"
	"time"

	"errors"

	"github.com/vprashar2929/integration-test/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

// TODO: Come up with better solution
// This is done so that unit test can work fine as retry.RetryOnConflict only works on real Kubernetes cluster
// we are using fake to build up mock client for UT
type Retryer interface {
	RetryOnConflict(backoff wait.Backoff, fn func() error) error
}

type DefaultRetryer struct{}

func (r *DefaultRetryer) RetryOnConflict(backoff wait.Backoff, fn func() error) error {
	return retry.RetryOnConflict(backoff, fn)
}

var retryer Retryer = &DefaultRetryer{}

var (
	ErrNoNamespace       = errors.New("error no namespace provided")
	ErrListingService    = errors.New("error listing services in namespace")
	ErrNoService         = errors.New("error no service in namespace")
	ErrServiceNotHealthy = errors.New("error service not in healthy state")
	ErrInvalidInterval   = errors.New("error interval or timeout is invalid")
	ErrServiceFailed     = errors.New("error service test validation failed")
)

func getService(namespace string, clientset kubernetes.Interface) (*corev1.ServiceList, error) {
	if len(namespace) == 0 {
		return nil, ErrNoNamespace
	}
	service, err := clientset.CoreV1().Services(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logger.AppLog.LogError("error listing services in namespace %s: %v\n", namespace, err)
		return nil, ErrListingService
	}
	if len(service.Items) == 0 {
		logger.AppLog.LogWarning("cannot find service in the namespace: %s\n", namespace)
		return nil, ErrNoService
	}
	return service, nil
}
func storeServicesByNamespace(namespaces []string, clientset kubernetes.Interface) (map[string][]corev1.Service, error) {
	if len(namespaces) == 0 {
		return nil, ErrNoNamespace
	}
	servicesByNamespace := make(map[string][]corev1.Service)
	for _, namespace := range namespaces {
		logger.AppLog.LogInfo("Checking Service status inside namespace %s\n", namespace)

		serviceList, err := getService(namespace, clientset)
		if err != nil {
			return nil, err
		}
		servicesByNamespace[namespace] = serviceList.Items
	}
	return servicesByNamespace, nil
}
func checkServiceStatus(namespace string, service corev1.Service, clientset kubernetes.Interface) error {
	return retryer.RetryOnConflict(retry.DefaultRetry, func() error {
		updatedService, err := clientset.CoreV1().Services(namespace).Get(context.Background(), service.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		endpoint, err := clientset.CoreV1().Endpoints(namespace).Get(context.Background(), service.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		for _, port := range updatedService.Spec.Ports {
			for _, subset := range endpoint.Subsets {
				for _, endpointPort := range subset.Ports {
					if endpointPort.Name == port.Name && endpointPort.Port == port.Port && len(subset.Addresses) > 0 {
						logger.AppLog.LogInfo("service %s is available in namespace %s\n", service.Name, namespace)
						return nil
					}
				}
			}
		}
		return ErrServiceNotHealthy
	})
}
func validateServicesByNamespace(namespaces []string, serviceByNamespace map[string][]corev1.Service, clientset kubernetes.Interface, interval, timeout time.Duration) error {
	var err error
	if interval <= 0 || timeout <= 0 {
		return ErrInvalidInterval
	}
	for _, namespace := range namespaces {
		for _, service := range serviceByNamespace[namespace] {
			deadline := time.Now().Add(timeout)
			for time.Now().Before(deadline) {
				if err = checkServiceStatus(namespace, service, clientset); err == nil {
					break
				}
				time.Sleep(interval)
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout checking service status for %s in namespace %s, error: %v", service.Name, namespace, err)
			}
		}
	}
	return nil
}
func CheckServices(namespaces []string, clientset kubernetes.Interface, interval, timeout time.Duration) error {
	logger.AppLog.LogInfo("Begin Service validation")

	serviceByNamespace, err := storeServicesByNamespace(namespaces, clientset)
	if err != nil {
		if errors.Is(err, ErrNoService) {
			logger.AppLog.LogWarning("No service found in namespace. Skipping validations")
			return nil
		}
		return err
	}
	err = validateServicesByNamespace(namespaces, serviceByNamespace, clientset, interval, timeout)
	if err != nil {
		return err
	}

	logger.AppLog.LogInfo("End Service validation")
	return nil

}
