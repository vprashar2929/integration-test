package service

import (
	"context"
	"time"

	"errors"

	"github.com/vprashar2929/rhobs-test/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

var (
	ErrNoNamespace       = errors.New("error no namespace provided")
	ErrListingService    = errors.New("error listing services in namespace")
	ErrFindingService    = errors.New("error finding services in namespace")
	ErrNoService         = errors.New("error no service in namespace")
	ErrNamespaceEmpty    = errors.New("error namespace list empty")
	ErrServiceNotHealthy = errors.New("error namespace not in healthy state")
	ErrInvalidInterval   = errors.New("error interval or timeout is invalid")
	ErrServiceFailed     = errors.New("error service test validation failed")
)

func getService(namespace string, clientset kubernetes.Interface) (*corev1.ServiceList, error) {
	if len(namespace) <= 0 {
		return nil, ErrNoNamespace
	}
	service, err := clientset.CoreV1().Services(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logger.AppLog.LogError("error listing services in namespace %s: %v\n", namespace, err)
		return nil, ErrListingService
	}
	if len(service.Items) <= 0 {
		logger.AppLog.LogWarning("cannot find service in the namespace: %s\n", namespace)
		return nil, ErrFindingService
	}
	return service, nil
}
func storeServicesByNamespace(namespaces []string, clientset kubernetes.Interface) (map[string][]corev1.Service, error) {
	if len(namespaces) <= 0 {
		return nil, ErrNoNamespace
	}
	servicesByNamespace := make(map[string][]corev1.Service)
	for _, namespace := range namespaces {
		serviceList, err := getService(namespace, clientset)
		if err != nil {
			logger.AppLog.LogError("error occured while fetching the service. reason: %v\n", err)
		}
		if len(serviceList.Items) > 0 {
			servicesByNamespace[namespace] = serviceList.Items
		}
	}
	if len(servicesByNamespace) <= 0 {
		return nil, ErrNoService
	}
	return servicesByNamespace, nil
}
func checkServiceStatus(namespace string, service corev1.Service, clientset kubernetes.Interface) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		updatedService, err := clientset.CoreV1().Services(namespace).Get(context.Background(), service.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		for _, port := range updatedService.Spec.Ports {
			endpoint, err := clientset.CoreV1().Endpoints(namespace).Get(context.Background(), service.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			for _, subset := range endpoint.Subsets {
				for _, endpointPort := range subset.Ports {
					if endpointPort.Name == port.Name && endpointPort.Port == port.Port {
						if len(subset.Addresses) > 0 {
							logger.AppLog.LogInfo("service %s is available in namespace %s\n", service.Name, namespace)
							return nil
						}
					}
				}
			}
		}
		logger.AppLog.LogWarning("waiting for service %s to be available in namespace %s\n", service.Name, namespace)
		logger.AppLog.LogWarning("service %s is not available yet in namespace %s\n", service.Name, namespace)
		return ErrServiceNotHealthy
	})
	return err
}
func validateServicesByNamespace(namespaces []string, serviceByNamespace map[string][]corev1.Service, clientset kubernetes.Interface, interval, timeout time.Duration) error {
	var errList []error
	if len(namespaces) <= 0 {
		logger.AppLog.LogWarning("namespace list empty %v. no namespace provided. please provide atleast one namespace\n", namespaces)
		return ErrNamespaceEmpty
	}
	if len(serviceByNamespace) <= 0 {
		return ErrNoService
	}
	if interval.Seconds() <= 0 || timeout.Seconds() <= 0 {
		logger.AppLog.LogError("interval or timeout is invalid. please provide the valid interval or timeout duration\n")
		return ErrInvalidInterval
	}
	for _, namespace := range namespaces {
		for _, service := range serviceByNamespace[namespace] {
			err := wait.Poll(interval, timeout, func() (bool, error) {
				err := checkServiceStatus(namespace, service, clientset)
				if err != nil {
					return false, nil
				}
				return true, nil
			})
			if err != nil {
				logger.AppLog.LogError("error checking the service %s in namespace %s status: %v\n", service.Name, namespace, err)
				errList = append(errList, err)
			}
		}
	}
	if len(errList) != 0 {
		logger.AppLog.LogError("to many errors. service validation test's failed")
		return ErrServiceFailed
	}
	return nil
}
func CheckServices(namespaces []string, clientset kubernetes.Interface, interval, timeout time.Duration) error {
	logger.AppLog.LogInfo("Begin Service validation")
	serviceByNamespace, err := storeServicesByNamespace(namespaces, clientset)
	if err != nil {
		if errors.Is(err, ErrNoService) {
			logger.AppLog.LogWarning("no service found in namespace. skipping service validations")
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
