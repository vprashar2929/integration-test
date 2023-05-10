package pod

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func checkPodStatus(namespace, resname string, clientset *kubernetes.Clientset) error {
	podList, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("app=%s", resname)})
	if err != nil {
		return fmt.Errorf("cannot list pod's in namespace %s: %v", namespace, err)
	}
	for _, pod := range podList.Items {
		podLogs, err := clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).Stream(context.Background())
		if err != nil {
			return fmt.Errorf("cannot fetch pod's logs in namespace %s: %v", namespace, err)
		}
		defer podLogs.Close()
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		if err != nil {
			return err
		}
		podLog := buf.String()
		if strings.Contains(podLog, "Error") || strings.Contains(podLog, "Exception") || strings.Contains(podLog, "error") || strings.Contains(podLog, "exception") {
			return fmt.Errorf("pod %s has errors in its logs: \n%s", pod.Name, podLog)
		}
	}
	return nil
}
func GetPodStatus(namespace, resname string, clientset *kubernetes.Clientset) error {
	return checkPodStatus(namespace, resname, clientset)
}
