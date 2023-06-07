package pod

import (
	"context"
	"fmt"
	"log"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

func checkPodHealth(namespace string, labels labels.Selector, clientset kubernetes.Interface) error {
	tailline := int64(10)
	seconds := int64(300)
	errCount := 0
	if len(namespace) <= 0 {
		return fmt.Errorf("no namespace provided to check pod logs\n")
	}
	podList, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labels.String()})
	if err != nil {
		return fmt.Errorf("cannot list pod's inside namespace: %s: %v\n", namespace, err)
	}
	err = checkPodStatus(namespace, *podList, clientset)
	if err != nil {
		return err
	}
	log.Println("Checking for error's/exception's in pod logs")
	for _, pod := range podList.Items {
		if pod.Status.Phase != "Running" {
			return fmt.Errorf("pod: %s is not running inside namespace: %s\n", pod.Name, namespace)

		}
		for _, container := range pod.Spec.Containers {
			logs, err := clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{Container: container.Name, SinceSeconds: &seconds, TailLines: &tailline}).Do(context.Background()).Raw()
			if err != nil {
				return fmt.Errorf("cannot fetch container: %s log's inside pod: %s error: %v\n", container.Name, pod.Name, err)
			}
			for _, line := range strings.Split(string(logs), "\\n") {
				if strings.Contains(line, "error") || strings.Contains(line, "Error") || strings.Contains(line, "Exception") || strings.Contains(line, "exception") {
					log.Printf("container: %s inside pod: %s has errors in logs: \n%s", container.Name, pod.Name, line)
					errCount += 1
				}
			}
		}

	}
	if errCount > 0 {
		return fmt.Errorf("error checking the Pod Health in namespace %s\n", namespace)
	}
	return nil
}
func checkPodStatus(namespace string, podList corev1.PodList, clientset kubernetes.Interface) error {
	if len(namespace) <= 0 {
		return fmt.Errorf("no namespace provided to check pod logs\n")
	}
	if len(podList.Items) <= 0 {
		return fmt.Errorf("cannot found pod inside the namespace\n")
	}
	log.Println("Checking pod status")
	for _, pod := range podList.Items {
		log.Printf("pod name: %s", pod.Name)
		if pod.Status.Phase != "Running" {
			return fmt.Errorf("pod: %s is not running inside namespace: %s\n", pod.Name, namespace)
		}
		for _, container := range pod.Status.ContainerStatuses {
			if container.RestartCount >= 1 && container.State.Waiting != nil && container.LastTerminationState.Terminated != nil {
				return fmt.Errorf("pod: %s has restart count: %d\ncurrent state: message: %s, reason: %s \nlast state: message: %s, reason: %s\n", container.Name, container.RestartCount, container.State.Waiting.Message, container.State.Waiting.Reason, container.LastTerminationState.Terminated.Message, container.LastTerminationState.Terminated.Reason)
			}
		}
	}
	return nil
}
func GetPodStatus(namespace string, labels labels.Selector, clientset kubernetes.Interface) error {
	return checkPodHealth(namespace, labels, clientset)
}
