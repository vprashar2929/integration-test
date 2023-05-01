package statefulset

import (
	"context"
	"fmt"
	"log"
	"time"

	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

func getStatefulSet(namespace string, clientset *kubernetes.Clientset) *appsv1.StatefulSetList {
	statefulset, err := clientset.AppsV1().StatefulSets(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("error listing statefulsets in namespace %s: %v\n", namespace, err)
	}

	return statefulset
}
func storeStatefulsetsByNamespace(namespaces []string, clientset *kubernetes.Clientset) (map[string][]appsv1.StatefulSet, error) {
	statefulsetsByNamespace := make(map[string][]appsv1.StatefulSet)
	for _, namespace := range namespaces {
		StatefulSetList := getStatefulSet(namespace, clientset)
		// Store the statefulsets by namespace in the map
		statefulsetsByNamespace[namespace] = StatefulSetList.Items
	}
	return statefulsetsByNamespace, nil
}
func checkStatefulSetStatus(namespace string, statefulset appsv1.StatefulSet, clientset *kubernetes.Clientset) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		updatedStatefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(context.Background(), statefulset.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if updatedStatefulSet.Status.UpdatedReplicas == *statefulset.Spec.Replicas &&
			updatedStatefulSet.Status.Replicas == *statefulset.Spec.Replicas &&
			updatedStatefulSet.Status.AvailableReplicas == *statefulset.Spec.Replicas &&
			updatedStatefulSet.Status.ObservedGeneration >= statefulset.Generation {
			fmt.Printf("statefulset %s is available in namespace %s\n", statefulset.Name, namespace)
			return nil
		} else {
			for _, condition := range updatedStatefulSet.Status.Conditions {
				if condition.Type == "StatefulSetReplicasReady" && condition.Status == corev1.ConditionFalse {
					fmt.Printf("Reason: %v\n", condition.Reason)
					break
				}
			}
		}
		fmt.Printf("Waiting for statefulset %s to be available in namespace %s\n", statefulset.Name, namespace)
		return fmt.Errorf("statefulset %s is not available yet in namespace %s\n", statefulset.Name, namespace)
	})
	return err
}
func validateStatefulSetsByNamespace(namespaces []string, statefulsetsByNamespace map[string][]appsv1.StatefulSet, clientset *kubernetes.Clientset, interval, timeout time.Duration) {
	for _, namespace := range namespaces {
		for _, statefulset := range statefulsetsByNamespace[namespace] {
			err := wait.Poll(interval, timeout, func() (bool, error) {
				err := checkStatefulSetStatus(namespace, statefulset, clientset)
				if err != nil {
					return false, nil
				}
				return true, nil
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error checking the statefulset %s in namespace %s status: %v\n", statefulset.Name, namespace, err)
				continue
			}

		}
	}
}
func CheckStatefulSets(namespace []string, clientset *kubernetes.Clientset, interval, timeout time.Duration) {
	statefulsetsByNamespace, err := storeStatefulsetsByNamespace(namespace, clientset)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error storing statefulsets by namespaces: %v\n", err)
	}
	validateStatefulSetsByNamespace(namespace, statefulsetsByNamespace, clientset, interval, timeout)
}