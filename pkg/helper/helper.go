package helpertest

import (
	"context"
	"log"
	"strings"
	"time"

	"fmt"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	replicas       = int32(1)
	clientset      *kubernetes.Clientset
	TestNamespace1 = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testns1",
		},
	}
	TestNamespace2 = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testns2",
		},
	}
	TestNamespace3 = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testns3",
		},
	}
	TestDeployment1 = appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "testns1",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-app",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test-app",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "my-container",
							Image: "quay.io/observatorium/up",
						},
					},
				},
			},
		},
	}
	TestDeployment2 = appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "testns2",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-app",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test-app",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "my-container",
							Image: "quay.io/observatorium/up",
						},
					},
				},
			},
		},
	}
	TestStatefulSet1 = appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-statefulset",
			Namespace: "testns1",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-app",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test-app",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "my-container",
							Image: "quay.io/observatorium/up",
						},
					},
				},
			},
		},
	}
	TestFaultyDeployment = appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "testns3",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-app",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test-app",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "my-container",
							Image:   "quay.io/observatorium/up",
							Command: []string{"docker ps -a"},
						},
					},
				},
			},
		},
	}
	TestNegativeDeployment = appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-negative-deployment",
			Namespace: "testns1",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-negative-app",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test-negative-app",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "my-negative-container",
							Image: "quay.io/observatorium/up",
							Args:  []string{"--endpoint-write", "http://localhost:9090"},
						},
					},
				},
			},
		},
	}
	TestStatefulSet2 = appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-statefulset",
			Namespace: "testns2",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-app",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test-app",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "my-container",
							Image: "quay.io/observatorium/up",
						},
					},
				},
			},
		},
	}
	testDeployments  = []appsv1.Deployment{TestDeployment1, TestDeployment2}
	testNamespaces   = []corev1.Namespace{TestNamespace1, TestNamespace2, TestNamespace3}
	testStatefulSets = []appsv1.StatefulSet{TestStatefulSet1, TestStatefulSet2}
)

func SetupTestEnvironment(resType string) (*kubernetes.Clientset, error) {
	currentDir, err := filepath.Abs("../")
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}
	parentDir := filepath.Dir(currentDir)
	if err != nil {
		return nil, fmt.Errorf("cannot list the contents inside the directory")
	}
	kubeconfig := filepath.Join(parentDir, "kubeconfig")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("cannot build from config: %v", err)
	}
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("cannot create clienset using the provided config: %v", err)
	}
	for _, namespace := range testNamespaces {
		_, err = clientset.CoreV1().Namespaces().Create(context.Background(), &namespace, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("cannot create test namespace, reason: %v", err)
		}
		time.Sleep(5 * time.Second)
	}
	if strings.ToLower(resType) == "deployment" {
		for _, deployment := range testDeployments {
			_, err = clientset.AppsV1().Deployments(deployment.Namespace).Create(context.Background(), &deployment, metav1.CreateOptions{})
			if err != nil {
				return nil, fmt.Errorf("cannot create test-deployment: %s", deployment.Name)
			}
			err = CheckTestDeploymentStatus(deployment.Namespace, deployment.Name, clientset)
			if err != nil {
				return nil, fmt.Errorf("test deployment status check failed. reason: %v", err)
			}

		}
	}
	if strings.ToLower(resType) == "statefulsets" {
		for _, statefulset := range testStatefulSets {
			_, err = clientset.AppsV1().StatefulSets(statefulset.Namespace).Create(context.Background(), &statefulset, metav1.CreateOptions{})
			if err != nil {
				return nil, fmt.Errorf("cannot create test-statefulset: %s", statefulset.Name)
			}
			err = CheckTestStatefulSetStatus(statefulset.Namespace, statefulset.Name, clientset)
			if err != nil {
				return nil, fmt.Errorf("test statefulset status check failed. reason: %v", err)
			}
		}
	}
	return clientset, nil
}
func TeardownTestEnvironment() error {
	for _, namespace := range testNamespaces {
		err := clientset.CoreV1().Namespaces().Delete(context.Background(), namespace.Name, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("cannot delete namespace, reason: %v", err)
		}
	}
	time.Sleep(15 * time.Second)
	return nil
}

func CheckTestDeploymentStatus(namespace, deployment string, clientset kubernetes.Interface) error {
	dep, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), deployment, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("cannot get the deployment %v in namespace %v", deployment, namespace)
	}
	for dep.Status.ReadyReplicas != *dep.Spec.Replicas {
		log.Printf("waiting for deployment %v to be finished....", dep.Name)
		time.Sleep(5 * time.Second)
		dep, err = clientset.AppsV1().Deployments(namespace).Get(context.Background(), dep.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error checking the status of deployment %v in namespace %v", dep.Name, namespace)
		}
	}
	return nil
}
func CheckTestStatefulSetStatus(namespace, statefulset string, clientset kubernetes.Interface) error {
	sts, err := clientset.AppsV1().StatefulSets(namespace).Get(context.Background(), statefulset, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("cannot get the statefulset %v in namespace %v", statefulset, namespace)
	}
	for sts.Status.ReadyReplicas != *sts.Spec.Replicas {
		log.Printf("waiting for statefulset %v to be finished.....", sts.Name)
		time.Sleep(5 * time.Second)
		sts, err = clientset.AppsV1().StatefulSets(namespace).Get(context.Background(), sts.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error checking the status of statefulset %v in namespace %v", sts.Name, namespace)
		}
	}
	return nil
}
