package statefulset

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/vprashar2929/rhobs-test/pkg/pod"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

func getStatefulSet(namespace string, clientset kubernetes.Interface) (*appsv1.StatefulSetList, error) {
	statefulset, err := clientset.AppsV1().StatefulSets(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing statefulsets in namespace %s: %v\n", namespace, err)
	}
	return statefulset, nil
}
func storeStatefulSetsByNamespace(namespaces []string, clientset kubernetes.Interface) (map[string][]appsv1.StatefulSet, error) {
	if len(namespaces) <= 0 {
		return nil, fmt.Errorf("no namespace provided\n")
	}
	statefulsetsByNamespace := make(map[string][]appsv1.StatefulSet)
	for _, namespace := range namespaces {
		if namespace != "" {
			statefulSetList, err := getStatefulSet(namespace, clientset)
			if err != nil {
				return nil, err
			}
			if len(statefulSetList.Items) > 0 {
				// Store the statefulsets by namespace in the map
				statefulsetsByNamespace[namespace] = statefulSetList.Items
			} else {
				log.Printf("no statefulset found under the namespace: %s\n", namespace)
			}
		} else {
			log.Printf("invalid namespace provided: %s\n", namespace)
		}

	}
	if len(statefulsetsByNamespace) <= 0 {
		return nil, fmt.Errorf("there is no statefulset in provided namespace: %v\n", namespaces)
	}
	return statefulsetsByNamespace, nil
}
func checkStatefulSetStatus(namespace string, statefulset appsv1.StatefulSet, clientset kubernetes.Interface) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		updatedStatefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(context.Background(), statefulset.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if updatedStatefulSet.Status.UpdatedReplicas == *statefulset.Spec.Replicas &&
			updatedStatefulSet.Status.Replicas == *statefulset.Spec.Replicas &&
			updatedStatefulSet.Status.AvailableReplicas == *statefulset.Spec.Replicas &&
			updatedStatefulSet.Status.ObservedGeneration >= statefulset.Generation {
			log.Printf("statefulset %s is available in namespace %s\n", statefulset.Name, namespace)
			return nil
		} else {
			log.Printf("statefulset %s is not available in namespace %s. checking condition\n", statefulset.Name, namespace)
			for _, condition := range updatedStatefulSet.Status.Conditions {
				if condition.Type == "StatefulSetReplicasReady" && condition.Status == corev1.ConditionFalse {
					log.Printf("reason: %v\n", condition.Reason)
					break
				}
			}
		}
		log.Printf("waiting for statefulset %s to be available in namespace %s\n", statefulset.Name, namespace)
		return fmt.Errorf("statefulset %s is not in healthy state inside namespace %s\n", statefulset.Name, namespace)
	})
	return err
}
func validateStatefulSetsByNamespace(namespaces []string, statefulsetsByNamespace map[string][]appsv1.StatefulSet, clientset kubernetes.Interface, interval, timeout time.Duration) error {
	var stsErrorList []error
	var podErrList []error
	if len(namespaces) <= 0 {
		return fmt.Errorf("namespace list empty %v. no namespace provided. please provide atleast one namespace\n", namespaces)
	}
	if len(statefulsetsByNamespace) <= 0 {
		return fmt.Errorf("no statefulset found in namespace. skipping statefulset validations\n")
	}
	if interval.Seconds() <= 0 || timeout.Seconds() <= 0 {
		return fmt.Errorf("interval or timeout is invalid. please provide the valid interval or timeout duration\n")
	}
	for _, namespace := range namespaces {
		for _, statefulset := range statefulsetsByNamespace[namespace] {
			err := wait.Poll(interval, timeout, func() (bool, error) {
				err := checkStatefulSetStatus(namespace, statefulset, clientset)
				if err != nil {
					return false, err
				}
				return true, nil
			})
			if err != nil {
				log.Printf("error checking the statefulset %s in namespace %s reason: %v\n", statefulset.Name, namespace, err)
				stsErrorList = append(stsErrorList, err)
			}
			err = wait.Poll(interval, timeout, func() (bool, error) {
				err := pod.GetPodStatus(namespace, labels.SelectorFromSet(statefulset.Spec.Selector.MatchLabels), clientset)
				if err != nil {
					return false, err
				}
				return true, nil
			})
			if err != nil {
				log.Printf("error checking the pod logs for statefulset %s in namespace %s, reason: %v\n", statefulset.Name, namespace, err)
				podErrList = append(podErrList, err)
			}

		}
	}
	if len(stsErrorList) != 0 || len(podErrList) != 0 {
		return fmt.Errorf("to many errors. statefulsets validation test's failed\n")
	}
	return nil
}
func CheckStatefulSets(namespace []string, clientset kubernetes.Interface, interval, timeout time.Duration) error {
	statefulsetsByNamespace, err := storeStatefulSetsByNamespace(namespace, clientset)
	if err != nil {
		return err
	}
	err = validateStatefulSetsByNamespace(namespace, statefulsetsByNamespace, clientset, interval, timeout)
	if err != nil {
		return err
	}
	return nil
}
