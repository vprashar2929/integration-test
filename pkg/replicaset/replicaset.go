package replicaset

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
	ErrListingReplicaSet    = errors.New("err listing replicaset in namespace")
	ErrNoReplicaSet         = errors.New("no replicaset found in namespace")
	ErrNamespaceEmpty       = errors.New("namespace list empty")
	ErrReplicaSetNotHealthy = errors.New("replicaset not in healthy state")
	ErrInvalidInterval      = errors.New("invalid interval or timeout")
	ErrReplicaSetFailed     = errors.New("replicaset validation failed")
)

func getReplicaSet(namespace string, clientset kubernetes.Interface) (*appsv1.ReplicaSetList, error) {
	replicaset, err := clientset.AppsV1().ReplicaSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, ErrListingReplicaSet
	}
	if len(replicaset.Items) == 0 {
		return nil, ErrNoReplicaSet
	}
	return replicaset, nil
}
func storeReplicaSetsByNamespace(namespaces []string, clientset kubernetes.Interface) (map[string][]appsv1.ReplicaSet, error) {
	if len(namespaces) == 0 {
		return nil, ErrNamespaceEmpty
	}
	replicasetsByNamespace := make(map[string][]appsv1.ReplicaSet)
	for _, namespace := range namespaces {
		if namespace == "" {
			logger.AppLog.LogError("Invalid namespace provided")
			continue
		}
		replicaSetList, err := getReplicaSet(namespace, clientset)
		if err != nil {
			return nil, err
		}
		replicasetsByNamespace[namespace] = replicaSetList.Items
	}
	if len(replicasetsByNamespace) == 0 {
		return nil, ErrNoReplicaSet
	}
	return replicasetsByNamespace, nil
}
func checkReplicaSetsStatus(namespace string, replicaset appsv1.ReplicaSet, clientset kubernetes.Interface) error {
	return retryer.RetryOnConflict(retry.DefaultRetry, func() error {
		updatedReplicaSet, err := clientset.AppsV1().ReplicaSets(namespace).Get(context.TODO(), replicaset.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if updatedReplicaSet.Status.Replicas == *replicaset.Spec.Replicas &&
			updatedReplicaSet.Status.ObservedGeneration >= replicaset.Generation &&
			updatedReplicaSet.Status.ReadyReplicas == replicaset.Status.Replicas {
			logger.AppLog.LogInfo("replicaset %s is available in namespace %s\n", replicaset.Name, namespace)
			return nil
		}
		for _, condition := range updatedReplicaSet.Status.Conditions {
			if condition.Type == "ReplicaSetReplicaReady" && condition.Status == corev1.ConditionFalse {
				logger.AppLog.LogError("reason: %v\n", condition.Reason)
				return ErrReplicaSetNotHealthy
			}
		}
		return ErrReplicaSetNotHealthy
	})
}
func validateReplicaSetsByNamespace(namespaces []string, replicasetsByNamespace map[string][]appsv1.ReplicaSet, clientset kubernetes.Interface, interval, timeout time.Duration) error {
	var err error
	if interval <= 0 || timeout <= 0 {
		return ErrInvalidInterval
	}
	for _, namespace := range namespaces {
		for _, replicaset := range replicasetsByNamespace[namespace] {

			// check replicaset status
			deadline := time.Now().Add(timeout)
			for time.Now().Before(deadline) {
				if err = checkReplicaSetsStatus(namespace, replicaset, clientset); err == nil {
					break
				}
				time.Sleep(interval)
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout checking replicaset status for %s in namespace %s, error: %v", replicaset.Name, namespace, err)
			}
			// check pod status
			deadline = time.Now().Add(timeout)
			for time.Now().Before(deadline) {
				if err = pod.GetPodStatus(namespace, labels.SelectorFromSet(replicaset.Spec.Selector.MatchLabels), clientset); err == nil {
					break
				}
				time.Sleep(interval)
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout checking pod status for replicaset %s in namespace %s, error: %v", replicaset.Name, namespace, err)
			}
		}
	}
	return nil
}
func CheckReplicaSets(namespaces []string, clienset kubernetes.Interface, interval, timeout time.Duration) error {
	logger.AppLog.LogInfo("Begin ReplicaSet validation")
	replicaSetsByNamespace, err := storeReplicaSetsByNamespace(namespaces, clienset)
	if err != nil {
		if errors.Is(err, ErrNoReplicaSet) {
			logger.AppLog.LogWarning("No replicasets found. Skipping validations.")
			return nil
		}
		return err
	}
	if err = validateReplicaSetsByNamespace(namespaces, replicaSetsByNamespace, clienset, interval, timeout); err != nil {
		return err
	}
	logger.AppLog.LogInfo("End ReplicaSet validations")
	return nil
}
