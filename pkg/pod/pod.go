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

func checkPodStatus(namespace string, labels labels.Selector, clientset *kubernetes.Clientset) error {
	log.Println("Checking for error's/exception's in pod logs")
	tailline := int64(10)
	seconds := int64(300)
	podList, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labels.String()})
	if err != nil {
		return fmt.Errorf("cannot list pod's inside namespace: %s: %v", namespace, err)
	}
	for _, pod := range podList.Items {
		if pod.Status.Phase != "Running" {
			return fmt.Errorf("pod: %s is not running inside namespace: %s: %v", pod.Name, namespace, err)

		}
		for _, container := range pod.Spec.Containers {
			logs, err := clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{Container: container.Name, SinceSeconds: &seconds, TailLines: &tailline}).Do(context.Background()).Raw()
			if err != nil {
				return fmt.Errorf("cannot fetch container: %s log's inside pod: %s error: %v", container.Name, pod.Name, err)
			}
			for _, line := range strings.Split(string(logs), "\\n") {
				if strings.Contains(line, "error") || strings.Contains(line, "Error") || strings.Contains(line, "Exception") || strings.Contains(line, "exception") {
					return fmt.Errorf("container: %s inside pod: %s has errors in logs: \n%s", container.Name, pod.Name, line)
				}
			}
		}
	}
	return nil
}
func GetPodStatus(namespace string, labels labels.Selector, clientset *kubernetes.Clientset) error {
	return checkPodStatus(namespace, labels, clientset)
}
