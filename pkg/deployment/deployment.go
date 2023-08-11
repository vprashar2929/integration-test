package deployment

import (
	"context"
	"fmt"
	"time"

	"errors"

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

var retryer Retryer = &DefaultRetryer{}

var (
	ErrListingDeployment    = errors.New("error listing deployments in namespace")
	ErrNoDeployment         = errors.New("no deployments found inside namespace")
	ErrNoNamespace          = errors.New("no namespace provided")
	ErrDeploymentUnhealthy  = errors.New("deployment not in healthy state")
	ErrInvalidInterval      = errors.New("interval or timeout is invalid")
	ErrDeploymentValidation = errors.New("deployment validation failed")
)

func getDeployment(namespace string, clientset kubernetes.Interface) (*appsv1.DeploymentList, error) {
	deployment, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, ErrListingDeployment
	}
	if len(deployment.Items) == 0 {
		return nil, ErrNoDeployment
	}
	return deployment, nil
}

func storeDeploymentsByNamespace(namespaces []string, clientset kubernetes.Interface) (map[string][]appsv1.Deployment, error) {
	if len(namespaces) == 0 {
		return nil, ErrNoNamespace
	}
	deploymentsByNamespace := make(map[string][]appsv1.Deployment)
	for _, namespace := range namespaces {
		if namespace == "" {
			logger.AppLog.LogError("Invalid namespace provided.")
			continue
		}
		deploymentList, err := getDeployment(namespace, clientset)
		if err != nil {
			return nil, err
		}
		deploymentsByNamespace[namespace] = deploymentList.Items
	}

	if len(deploymentsByNamespace) == 0 {
		return nil, ErrNoDeployment
	}
	return deploymentsByNamespace, nil
}

func checkDeploymentStatus(namespace string, deployment appsv1.Deployment, clientset kubernetes.Interface) error {
	return retryer.RetryOnConflict(retry.DefaultRetry, func() error {
		updatedDeployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deployment.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if updatedDeployment.Status.UpdatedReplicas == *deployment.Spec.Replicas &&
			updatedDeployment.Status.Replicas == *deployment.Spec.Replicas &&
			updatedDeployment.Status.AvailableReplicas == *deployment.Spec.Replicas &&
			updatedDeployment.Status.ObservedGeneration >= deployment.Generation {
			return nil
		}

		for _, condition := range updatedDeployment.Status.Conditions {
			if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionFalse {
				return ErrDeploymentUnhealthy
			}
		}
		return nil
	})
}

func validateDeploymentsByNamespace(namespaces []string, deploymentsByNamespace map[string][]appsv1.Deployment, clientset kubernetes.Interface, interval, timeout time.Duration) error {
	var err error
	if interval <= 0 || timeout <= 0 {
		return ErrInvalidInterval
	}
	for _, namespace := range namespaces {
		for _, deployment := range deploymentsByNamespace[namespace] {

			// check deployment status
			deadline := time.Now().Add(timeout)
			for time.Now().Before(deadline) {
				if err = checkDeploymentStatus(namespace, deployment, clientset); err == nil {
					break
				}
				time.Sleep(interval)
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout checking deployment status for %s in namespace %s, error: %v ", deployment.Name, namespace, err)
			}

			// check pod status
			deadline = time.Now().Add(timeout)
			for time.Now().Before(deadline) {
				if err = pod.GetPodStatus(namespace, labels.SelectorFromSet(deployment.Spec.Selector.MatchLabels), clientset); err == nil {
					break
				}
				time.Sleep(interval)
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout checking pod status for deployment %s in namespace %s, error: %v", deployment.Name, namespace, err)
			}

		}
	}

	return nil
}

func CheckDeployments(namespaces []string, clientset kubernetes.Interface, interval, timeout time.Duration) error {
	logger.AppLog.LogInfo("Begin Deployment validation")

	deploymentsByNamespace, err := storeDeploymentsByNamespace(namespaces, clientset)
	if err != nil {
		if errors.Is(err, ErrNoDeployment) {
			logger.AppLog.LogWarning("No deployments found. Skipping validations.")
			return nil
		}
		return err
	}

	if err := validateDeploymentsByNamespace(namespaces, deploymentsByNamespace, clientset, interval, timeout); err != nil {
		return err
	}

	logger.AppLog.LogInfo("End Deployment validation")
	return nil
}
