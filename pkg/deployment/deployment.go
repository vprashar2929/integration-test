package deployment

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

func getDeployment(namespace string, clientset kubernetes.Interface) (*appsv1.DeploymentList, error) {
	deployment, err := clientset.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing deployments in namespace %s: %v\n", namespace, err)
	}

	return deployment, nil
}
func storeDeploymentsByNamespace(namespaces []string, clientset kubernetes.Interface) (map[string][]appsv1.Deployment, error) {
	if len(namespaces) <= 0 {
		return nil, fmt.Errorf("no namespace provided")
	}
	deploymentsByNamespace := make(map[string][]appsv1.Deployment)
	for _, namespace := range namespaces {
		if namespace != "" {
			deploymentList, err := getDeployment(namespace, clientset)
			if err != nil {
				return nil, err
			}
			if len(deploymentList.Items) > 0 {
				// Store the deployments by namespace in the map
				deploymentsByNamespace[namespace] = deploymentList.Items
			} else {
				log.Printf("no deployment found under the namespace: %s\n", namespace)
			}
		} else {
			log.Printf("invalid namespace provided: %s\n", namespace)
		}

	}
	if len(deploymentsByNamespace) <= 0 {
		return nil, fmt.Errorf("there is no deployment in provided namespaces: %v\n", namespaces)
	}
	return deploymentsByNamespace, nil
}
func checkDeploymentStatus(namespace string, deployment appsv1.Deployment, clientset kubernetes.Interface) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		updatedDeployment, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), deployment.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if updatedDeployment.Status.UpdatedReplicas == *deployment.Spec.Replicas &&
			updatedDeployment.Status.Replicas == *deployment.Spec.Replicas &&
			updatedDeployment.Status.AvailableReplicas == *deployment.Spec.Replicas &&
			updatedDeployment.Status.ObservedGeneration >= deployment.Generation {
			log.Printf("deployment %s is available in namespace %s\n", deployment.Name, namespace)
			return nil
		} else {
			log.Printf("deployment %s is not available in namespace %s. Checking condition\n", deployment.Name, namespace)
			for _, condition := range updatedDeployment.Status.Conditions {
				if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionFalse {
					log.Printf("reason: %v\n", condition.Reason)
					break
				}
			}
		}
		log.Printf("waiting for deployment %s to be available in namespace %s\n", deployment.Name, namespace)
		return fmt.Errorf("deployment %v is not in healthy state inside namespace %v\n", deployment.Name, namespace)
	})
	return err
}
func validateDeploymentsByNamespace(namespaces []string, deploymentsByNamespace map[string][]appsv1.Deployment, clientset kubernetes.Interface, interval, timeout time.Duration) error {
	var depErrList []error
	var podErrList []error
	if len(namespaces) <= 0 {
		return fmt.Errorf("namespace list empty %v. no namespace provided. please provide atleast one namespace\n", namespaces)
	}
	if len(deploymentsByNamespace) <= 0 {
		return fmt.Errorf("no deployment found in namespaces. skipping deployment validations\n")
	}
	if interval.Seconds() <= 0 || timeout.Seconds() <= 0 {
		return fmt.Errorf("interval or timeout is invalid. please provide the valid interval or timeout duration\n")
	}
	for _, namespace := range namespaces {
		for _, deployment := range deploymentsByNamespace[namespace] {
			err := wait.Poll(interval, timeout, func() (bool, error) {
				err := checkDeploymentStatus(namespace, deployment, clientset)
				if err != nil {
					return false, err
				}
				return true, nil

			})
			if err != nil {
				log.Printf("error checking the deployment %s in namespace %s reason: %v\n", deployment.Name, namespace, err)
				depErrList = append(depErrList, err)
			}
			err = wait.Poll(interval, timeout, func() (bool, error) {
				err := pod.GetPodStatus(namespace, labels.SelectorFromSet(deployment.Spec.Selector.MatchLabels), clientset)
				if err != nil {
					return false, err
				}
				return true, nil
			})
			if err != nil {
				log.Printf("error checking the pod logs of deployment %s in namespace %s, reason: %v\n", deployment.Name, namespace, err)
				podErrList = append(podErrList, err)
			}

		}
	}
	if len(depErrList) != 0 || len(podErrList) != 0 {
		return fmt.Errorf("to many errors. deployment validation test's failed\n")
	}
	return nil
}
func CheckDeployments(namespace []string, clientset kubernetes.Interface, interval, timeout time.Duration) error {
	deploymentsByNamespace, err := storeDeploymentsByNamespace(namespace, clientset)
	if err != nil {
		return err
	}
	err = validateDeploymentsByNamespace(namespace, deploymentsByNamespace, clientset, interval, timeout)
	if err != nil {
		return err
	}
	return nil
}
