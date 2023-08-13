package statefulset

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vprashar2929/integration-test/pkg/logger"
	"github.com/vprashar2929/integration-test/pkg/pod"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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

var (
	retryer Retryer = &DefaultRetryer{}
)

var (
	ErrListingStatefulSet    = errors.New("error listing statefulsets in namespace")
	ErrNoStatefulSet         = errors.New("no statefulset found in namespace")
	ErrNamespaceEmpty        = errors.New("namespace list empty")
	ErrStatefulSetNotHealthy = errors.New("statefulset not in healthy state")
	ErrInvalidInterval       = errors.New("invalid interval or timeout")
	ErrStatefulSetFailed     = errors.New("statefulset validation failed")
)

func getStatefulSet(namespace string, clientset kubernetes.Interface) (*appsv1.StatefulSetList, error) {
	statefulset, err := clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, ErrListingStatefulSet
	}
	if len(statefulset.Items) == 0 {
		return nil, ErrNoStatefulSet
	}
	return statefulset, nil
}
func storeStatefulSetsByNamespace(namespaces []string, clientset kubernetes.Interface) (map[string][]appsv1.StatefulSet, error) {
	if len(namespaces) == 0 {
		return nil, ErrNamespaceEmpty
	}
	statefulsetsByNamespace := make(map[string][]appsv1.StatefulSet)
	for _, namespace := range namespaces {
		if namespace == "" {
			logger.AppLog.LogError("Invalid namespace provided")
			continue
		}
		statefulSetList, err := getStatefulSet(namespace, clientset)
		if err != nil {
			return nil, err
		}
		statefulsetsByNamespace[namespace] = statefulSetList.Items
	}
	if len(statefulsetsByNamespace) == 0 {
		return nil, ErrNoStatefulSet
	}

	return statefulsetsByNamespace, nil
}

func checkStatefulSetStatus(namespace string, statefulset appsv1.StatefulSet, clientset kubernetes.Interface) error {
	return retryer.RetryOnConflict(retry.DefaultRetry, func() error {
		updatedStatefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), statefulset.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if updatedStatefulSet.Status.UpdatedReplicas == *statefulset.Spec.Replicas &&
			updatedStatefulSet.Status.Replicas == *statefulset.Spec.Replicas &&
			updatedStatefulSet.Status.CurrentReplicas == *statefulset.Spec.Replicas &&
			updatedStatefulSet.Status.ObservedGeneration >= statefulset.Generation {
			logger.AppLog.LogInfo("statefulset %s is available in namespace %s\n", statefulset.Name, namespace)
			return nil
		}
		for _, condition := range updatedStatefulSet.Status.Conditions {
			if condition.Type == "StatefulSetReplicasReady" && condition.Status == corev1.ConditionFalse {
				logger.AppLog.LogError("reason: %v\n", condition.Reason)
				return ErrStatefulSetNotHealthy
			}
		}
		return ErrStatefulSetNotHealthy
	})
}
func validateStatefulSetsByNamespace(namespaces []string, statefulsetsByNamespace map[string][]appsv1.StatefulSet, clientset kubernetes.Interface, interval, timeout time.Duration) error {
	var err error

	if interval <= 0 || timeout <= 0 {
		return ErrInvalidInterval
	}
	for _, namespace := range namespaces {
		for _, statefulset := range statefulsetsByNamespace[namespace] {
			// check statefulset status

			deadline := time.Now().Add(timeout)
			for time.Now().Before(deadline) {
				if err = checkStatefulSetStatus(namespace, statefulset, clientset); err == nil {
					break
				}
				time.Sleep(interval)
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout checking statefulset status for %s in namespace %s, error: %v", statefulset.Name, namespace, err)
			}

			// check pods status

			deadline = time.Now().Add(timeout)
			for time.Now().Before(deadline) {
				if err = pod.GetPodStatus(namespace, labels.SelectorFromSet(statefulset.Spec.Selector.MatchLabels), clientset); err == nil {
					break
				}
				time.Sleep(interval)
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout checking pod status for statefulset %s in namespace %s, error: %v", statefulset.Name, namespace, err)
			}
		}
	}

	return nil
}
func CheckStatefulSets(namespace []string, clientset kubernetes.Interface, interval, timeout time.Duration) error {
	logger.AppLog.LogInfo("Begin StatefulSet validation")

	statefulsetsByNamespace, err := storeStatefulSetsByNamespace(namespace, clientset)
	if err != nil {
		if errors.Is(err, ErrNoStatefulSet) {
			logger.AppLog.LogWarning("No statefulsets found. Skipping validations.")
			return nil
		}
		return err
	}
	if err = validateStatefulSetsByNamespace(namespace, statefulsetsByNamespace, clientset, interval, timeout); err != nil {
		return err
	}

	logger.AppLog.LogInfo("End StatefulSet validation")
	return nil
}
