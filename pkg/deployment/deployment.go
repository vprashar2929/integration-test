package deployment

import (
	"context"
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

func getDeployment(namespace string, clientset *kubernetes.Clientset) *appsv1.DeploymentList {
	deployment, err := clientset.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("error listing deployments in namespace %s: %v\n", namespace, err)
	}

	return deployment
}
func storeDeploymentsByNamespace(namespaces []string, clientset *kubernetes.Clientset) map[string][]appsv1.Deployment {
	deploymentsByNamespace := make(map[string][]appsv1.Deployment)
	for _, namespace := range namespaces {
		deploymentList := getDeployment(namespace, clientset)
		// Store the deployments by namespace in the map
		deploymentsByNamespace[namespace] = deploymentList.Items
	}
	return deploymentsByNamespace
}
func checkDeploymentStatus(namespace string, deployment appsv1.Deployment, clientset *kubernetes.Clientset) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		updatedDeployment, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), deployment.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if updatedDeployment.Status.UpdatedReplicas == *deployment.Spec.Replicas &&
			updatedDeployment.Status.Replicas == *deployment.Spec.Replicas &&
			updatedDeployment.Status.AvailableReplicas == *deployment.Spec.Replicas &&
			updatedDeployment.Status.ObservedGeneration >= deployment.Generation {
			log.Printf("Deployment %s is available in namespace %s\n", deployment.Name, namespace)
			return pod.GetPodStatus(namespace, labels.SelectorFromSet(deployment.Spec.Selector.MatchLabels), clientset)
		} else {
			for _, condition := range updatedDeployment.Status.Conditions {
				if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionFalse {
					log.Printf("Reason: %v\n", condition.Reason)
					break
				}
			}
		}
		log.Printf("Waiting for deployment %s to be available in namespace %s\n", deployment.Name, namespace)
		return pod.GetPodStatus(namespace, labels.SelectorFromSet(deployment.Spec.Selector.MatchLabels), clientset)
	})
	return err
}
func validateDeploymentsByNamespace(namespaces []string, deploymentsByNamespace map[string][]appsv1.Deployment, clientset *kubernetes.Clientset, interval, timeout time.Duration) {
	var errList []error
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
				log.Printf("Error checking the deployment %s in namespace %s Reason: %v\n", deployment.Name, namespace, err)
				errList = append(errList, err)
				continue
			}

		}
	}
	if len(errList) != 0 {
		log.Fatal("To many errors. Deployment validation test's failed!!!!!!!!!!")
	}
}
func CheckDeployments(namespace []string, clientset *kubernetes.Clientset, interval, timeout time.Duration) {
	deploymentsByNamespace := storeDeploymentsByNamespace(namespace, clientset)
	validateDeploymentsByNamespace(namespace, deploymentsByNamespace, clientset, interval, timeout)
}
