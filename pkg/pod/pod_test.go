package pod

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"time"

	helper "github.com/vprashar2929/rhobs-test/pkg/helper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

var (
	clientset *kubernetes.Clientset
)

func TestCheckPodHealthNegative(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	type args struct {
		namespace string
		labels    labels.Selector
		clientset kubernetes.Interface
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test with invalid namespace",
			args: args{
				namespace: "",
				labels:    labels.SelectorFromSet(helper.TestDeployment1.Spec.Selector.MatchLabels),
				clientset: clientset,
			},
		},
		{
			name: "Test with no pod in a namespace",
			args: args{
				namespace: "testns3",
				labels:    labels.SelectorFromSet(helper.TestDeployment2.Spec.Selector.MatchLabels),
				clientset: clientset,
			},
		},
		{
			name: "Test with faulty deployment",
			args: args{
				namespace: "testns3",
				labels:    labels.SelectorFromSet(helper.TestFaultyDeployment.Spec.Selector.MatchLabels),
				clientset: clientset,
			},
		},
		{
			name: "Test with negative deployment",
			args: args{
				namespace: "testns1",
				labels:    labels.SelectorFromSet(helper.TestNegativeDeployment.Spec.Selector.MatchLabels),
				clientset: clientset,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch name := tt.name; name {
			case "Test with invalid namespace":
				err := checkPodHealth(tt.args.namespace, tt.args.labels, tt.args.clientset)
				if err == nil {
					t.Errorf("checkPodHealth() should return error. got: %v", err)
				}
			case "Test with no pod in a namespace":
				err := checkPodHealth(tt.args.namespace, tt.args.labels, tt.args.clientset)
				if err == nil {
					t.Errorf("checkPodHealth() should return error. got: %v", err)
				}
			case "Test with faulty deployment":
				_, err := clientset.AppsV1().Deployments(helper.TestFaultyDeployment.Namespace).Create(context.Background(), &helper.TestFaultyDeployment, metav1.CreateOptions{})
				if err != nil {
					t.Errorf("failed to create faulty deployment %v, reason: %v", helper.TestFaultyDeployment.Name, err)
				}
				time.Sleep(15 * time.Second)
				err = checkPodHealth(tt.args.namespace, tt.args.labels, tt.args.clientset)
				if err == nil {
					t.Errorf("checkPodHealth() should return error. got: %v", err)
				}
			case "Test with negative deployment":
				_, err := clientset.AppsV1().Deployments(helper.TestNegativeDeployment.Namespace).Create(context.Background(), &helper.TestNegativeDeployment, metav1.CreateOptions{})
				if err != nil {
					t.Errorf("failed to create negative deployment %v, reason: %v", helper.TestNegativeDeployment.Name, err)
				}
				time.Sleep(15 * time.Second)
				err = checkPodHealth(tt.args.namespace, tt.args.labels, tt.args.clientset)
				if err == nil {
					t.Errorf("checkPodHealth() should return error. got: %v", err)
				}
			}

		})
	}
}
func TestCheckPodHealthPositive(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	type args struct {
		namespace string
		labels    labels.Selector
		clientset kubernetes.Interface
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test with valid namespace",
			args: args{
				namespace: "testns1",
				labels:    labels.SelectorFromSet(helper.TestDeployment1.Spec.Selector.MatchLabels),
				clientset: clientset,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch name := tt.name; name {
			case "Test with valid namespace":
				err := checkPodHealth(tt.args.namespace, tt.args.labels, tt.args.clientset)
				if err != nil {
					t.Errorf("checkPodHealth() should not return error. got: %v", err)
				}
			}
		})
	}
}
func TestCheckPodStatusNegative(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	type args struct {
		namespace string
		podList   corev1.PodList
		clientset kubernetes.Interface
	}
	faultyPodSpec := corev1.Pod{Spec: helper.TestFaultyDeployment.Spec.Template.Spec}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test with invalid namespace",
			args: args{
				namespace: "",
				podList:   corev1.PodList{},
				clientset: clientset,
			},
		},
		{
			name: "Test with no pod in a namespace",
			args: args{
				namespace: "testns3",
				podList:   corev1.PodList{Items: []corev1.Pod{}},
				clientset: clientset,
			},
		},
		{
			name: "Test with faulty deployment",
			args: args{
				namespace: "testns3",
				podList:   corev1.PodList{Items: []corev1.Pod{faultyPodSpec}},
				clientset: clientset,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch name := tt.name; name {
			case "Test with invalid namespace":
				err := checkPodStatus(tt.args.namespace, tt.args.podList, tt.args.clientset)
				if err == nil {
					t.Errorf("checkPodHealth() should return error. got: %v", err)
				}
			case "Test with no pod in a namespace":
				err := checkPodStatus(tt.args.namespace, tt.args.podList, tt.args.clientset)
				if err == nil {
					t.Errorf("checkPodHealth() should return error. got: %v", err)
				}
			case "Test with faulty deployment":
				err := checkPodStatus(tt.args.namespace, tt.args.podList, tt.args.clientset)
				if err == nil {
					t.Errorf("checkPodHealth() should return error. got: %v", err)
				}
			}

		})
	}
}
func TestCheckPodStatusPositive(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	type args struct {
		namespace string
		podList   corev1.PodList
		clientset kubernetes.Interface
	}
	podList, err := clientset.CoreV1().Pods(helper.TestDeployment1.Namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labels.SelectorFromSet(helper.TestDeployment1.Spec.Selector.MatchLabels).String()})
	if err != nil {
		t.Errorf("cannot list pod's inside namespace: %s: %v", helper.TestDeployment1.Namespace, err)
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test with valid namespace",
			args: args{
				namespace: "testns1",
				podList:   *podList,
				clientset: clientset,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch name := tt.name; name {
			case "Test with valid namespace":
				err := checkPodStatus(tt.args.namespace, tt.args.podList, tt.args.clientset)
				if err != nil {
					t.Errorf("checkPodHealth() should not return error. got: %v", err)
				}
			}

		})
	}
}
func TestMain(m *testing.M) {
	cls, err := helper.SetupTestEnvironment("deployment")
	clientset = cls

	if err != nil {
		fmt.Printf("issue while setting up environment to run the tests. reason: %v", err)
		os.Exit(1)
	}
	exitCode := m.Run()
	err = helper.TeardownTestEnvironment()
	if err != nil {
		fmt.Printf("issue while destroying the environment. reason: %v", err)
		os.Exit(1)
	}
	os.Exit(exitCode)
}
