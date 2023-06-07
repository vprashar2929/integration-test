package service

import (
	"context"
	"fmt"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

func getService(namespace string, clientset *kubernetes.Clientset) (*corev1.ServiceList, error) {
	if len(namespace) <= 0 {
		return nil, fmt.Errorf("no namespace provided\n")
	}
	service, err := clientset.CoreV1().Services(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing services in namespace %s: %v\n", namespace, err)
	}
	if len(service.Items) <= 0 {
		return nil, fmt.Errorf("cannot find service in the namespace: %s\n", namespace)
	}
	return service, nil
}
func storeServicesByNamespace(namespaces []string, clientset *kubernetes.Clientset) (map[string][]corev1.Service, error) {
	if len(namespaces) <= 0 {
		return nil, fmt.Errorf("no namespace provided. please provide atleast one namespace\n")
	}
	servicesByNamespace := make(map[string][]corev1.Service)
	for _, namespace := range namespaces {
		serviceList, err := getService(namespace, clientset)
		if err != nil {
			log.Printf("error occured while fetching the service. reason: %v\n", err)
		}
		if len(serviceList.Items) > 0 {
			servicesByNamespace[namespace] = serviceList.Items
		}
	}
	if len(servicesByNamespace) <= 0 {
		return nil, fmt.Errorf("no service found in provided namespaces")
	}
	return servicesByNamespace, nil
}
func checkServiceStatus(namespace string, service corev1.Service, clientset *kubernetes.Clientset) error {
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
							log.Printf("service %s is available in namespace %s\n", service.Name, namespace)
							return nil
						}
					}

				}
			}
		}
		log.Printf("waiting for service %s to be available in namespace %s\n", service.Name, namespace)
		return fmt.Errorf("service %s is not available yet in namespace %s\n", service.Name, namespace)
	})
	return err
}
func validateServicesByNamespace(namespaces []string, serviceByNamespace map[string][]corev1.Service, clientset *kubernetes.Clientset, interval, timeout time.Duration) error {
	var errList []error
	if len(namespaces) <= 0 {
		return fmt.Errorf("namespace list empty %v. no namespace provided. please provide atleast one namespace\n", namespaces)
	}
	if len(serviceByNamespace) <= 0 {
		return fmt.Errorf("no service found in namespace. skipping service validations\n")
	}
	if interval.Seconds() <= 0 || timeout.Seconds() <= 0 {
		return fmt.Errorf("interval or timeout is invalid. please provide the valid interval or timeout duration\n")
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
				log.Printf("error checking the service %s in namespace %s status: %v\n", service.Name, namespace, err)
				errList = append(errList, err)
			}
		}
	}
	if len(errList) != 0 {
		return fmt.Errorf("to many errors. service validation test's failed\n")
	}
	return nil
}
func CheckServices(namespaces []string, clientset *kubernetes.Clientset, interval, timeout time.Duration) error {
	serviceByNamespace, err := storeServicesByNamespace(namespaces, clientset)
	if err != nil {
		return err
	}
	err = validateServicesByNamespace(namespaces, serviceByNamespace, clientset, interval, timeout)
	if err != nil {
		return err
	}
	return nil

}
