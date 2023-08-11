package daemonset

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
	ErrListingDaemonSet    = errors.New("err listing daemonset in namespace")
	ErrNoDaemonSet         = errors.New("no daemonset found in namespace")
	ErrNamespaceEmpty      = errors.New("namespace list empty")
	ErrDaemonSetNotHealthy = errors.New("daemonset not in healthy state")
	ErrInvalidInterval     = errors.New("invalid interval or timeout")
	ErrDaemonSetFailed     = errors.New("daemonset validation failed")
)

func getDaemonSet(namespace string, clientset kubernetes.Interface) (*appsv1.DaemonSetList, error) {
	daemonset, err := clientset.AppsV1().DaemonSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, ErrListingDaemonSet
	}
	if len(daemonset.Items) == 0 {
		return nil, ErrNoDaemonSet
	}
	return daemonset, nil
}
func storeDaemonSetsByNamespace(namespaces []string, clientset kubernetes.Interface) (map[string][]appsv1.DaemonSet, error) {
	if len(namespaces) == 0 {
		return nil, ErrNamespaceEmpty
	}
	daemonsetsByNamespace := make(map[string][]appsv1.DaemonSet)
	for _, namespace := range namespaces {
		if namespace == "" {
			logger.AppLog.LogError("Invalid namespace provided")
			continue
		}
		daemonSetList, err := getDaemonSet(namespace, clientset)
		if err != nil {
			return nil, err
		}
		daemonsetsByNamespace[namespace] = daemonSetList.Items
	}
	if len(daemonsetsByNamespace) == 0 {
		return nil, ErrNoDaemonSet
	}
	return daemonsetsByNamespace, nil
}
func checkDaemonSetsStatus(namespace string, daemonset appsv1.DaemonSet, clientset kubernetes.Interface) error {
	return retryer.RetryOnConflict(retry.DefaultRetry, func() error {
		updatedDaemonSet, err := clientset.AppsV1().DaemonSets(namespace).Get(context.TODO(), daemonset.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if updatedDaemonSet.Status.DesiredNumberScheduled == daemonset.Status.NumberAvailable &&
			updatedDaemonSet.Status.DesiredNumberScheduled == daemonset.Status.CurrentNumberScheduled &&
			updatedDaemonSet.Status.NumberUnavailable < 0 {
			logger.AppLog.LogInfo("daemonset %s is available in namespace %s\n", daemonset.Name, namespace)
			return nil
		}
		for _, condition := range updatedDaemonSet.Status.Conditions {
			if condition.Type == "DaemonSetDaemonReady" && condition.Status == corev1.ConditionFalse {
				logger.AppLog.LogError("reason: %v\n", condition.Reason)
				return ErrDaemonSetNotHealthy
			}
		}
		return ErrDaemonSetNotHealthy
	})
}
func validateDaemonSetsByNamespace(namespaces []string, daemonsetsByNamespace map[string][]appsv1.DaemonSet, clientset kubernetes.Interface, interval, timeout time.Duration) error {
	var err error
	if interval <= 0 || timeout <= 0 {
		return ErrInvalidInterval
	}
	for _, namespace := range namespaces {
		for _, daemonset := range daemonsetsByNamespace[namespace] {

			// check daemonset status
			deadline := time.Now().Add(timeout)
			for time.Now().Before(deadline) {
				if err = checkDaemonSetsStatus(namespace, daemonset, clientset); err == nil {
					break
				}
				time.Sleep(interval)
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout checking daemonset status for %s in namespace %s, error: %v", daemonset.Name, namespace, err)
			}
			// check pod status
			deadline = time.Now().Add(timeout)
			for time.Now().Before(deadline) {
				if err = pod.GetPodStatus(namespace, labels.SelectorFromSet(daemonset.Spec.Selector.MatchLabels), clientset); err == nil {
					break
				}
				time.Sleep(interval)
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout checking pod status for daemonset %s in namespace %s, error: %v", daemonset.Name, namespace, err)
			}
		}
	}
	return nil
}
func CheckDaemonSets(namespaces []string, clienset kubernetes.Interface, interval, timeout time.Duration) error {
	logger.AppLog.LogInfo("Begin DaemonSet validation")
	daemonSetsByNamespace, err := storeDaemonSetsByNamespace(namespaces, clienset)
	if err != nil {
		if errors.Is(err, ErrNoDaemonSet) {
			logger.AppLog.LogWarning("No daemonsets found. Skipping validations.")
			return nil
		}
		return err
	}
	if err = validateDaemonSetsByNamespace(namespaces, daemonSetsByNamespace, clienset, interval, timeout); err != nil {
		return err
	}
	logger.AppLog.LogInfo("End DaemonSet validations")
	return nil
}
