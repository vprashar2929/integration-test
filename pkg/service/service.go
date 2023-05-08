package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

func getService(namespace string, clientset *kubernetes.Clientset) *corev1.ServiceList {
	service, err := clientset.CoreV1().Services(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("error listing services in namespace %s: %v\n", namespace, err)
	}
	return service
}
func storeServicesByNamespace(namespaces []string, clientset *kubernetes.Clientset) map[string][]corev1.Service {
	servicesByNamespace := make(map[string][]corev1.Service)
	for _, namespace := range namespaces {
		serviceList := getService(namespace, clientset)
		servicesByNamespace[namespace] = serviceList.Items
	}
	return servicesByNamespace
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
							fmt.Printf("Service %s is available in namespace %s\n", service.Name, namespace)
							return nil
						}
					}

				}
			}
		}
		fmt.Printf("Waiting for service %s to be available in namespace %s\n", service.Name, namespace)
		return fmt.Errorf("service %s is not available yet in namespace %s\n", service.Name, namespace)
	})
	return err
}
func validateServicesByNamespace(namespaces []string, serviceByNamespace map[string][]corev1.Service, clientset *kubernetes.Clientset, interval, timeout time.Duration) {
	var errList []error
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
				fmt.Fprintf(os.Stderr, "Error checking the service %s in namespace %s status: %v\n", service.Name, namespace, err)
				errList = append(errList, err)
				continue
			}
		}
	}
	if len(errList) != 0 {
		log.Fatalf("To many errors. Service validation test's failed!!!!!!!!!!!!")
	}
}
func CheckServices(namespaces []string, clientset *kubernetes.Clientset, interval, timeout time.Duration) {
	serviceByNamespace := storeServicesByNamespace(namespaces, clientset)
	validateServicesByNamespace(namespaces, serviceByNamespace, clientset, interval, timeout)

}
