package deployment

import (
	"context"
	"io"
	"log"
	"os"

	"fmt"
	"testing"
	"time"

	helper "github.com/vprashar2929/rhobs-test/pkg/helper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	replicas  = int32(1)
	clientset *kubernetes.Clientset
)

func TestGetDeploymentPositive(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	type args struct {
		namespace string
		clientset kubernetes.Interface
	}
	tests := struct {
		name string
		args args
		want *appsv1.DeploymentList
	}{
		name: "Test with valid params",
		args: args{
			namespace: "testns1",
			clientset: clientset,
		},
		want: &appsv1.DeploymentList{
			Items: []appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-deployment",
						Namespace: "testns1",
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: &replicas,
					},
				},
			},
		},
	}
	t.Run(tests.name, func(t *testing.T) {
		got, err := getDeployment(tests.args.namespace, tests.args.clientset)
		if err != nil {
			t.Errorf("%v", err)
		}
		if len(got.Items) != 1 {
			t.Fatalf("getDeployment() returned wrong number of deployments: %d, expected: %d\n", len(got.Items), len(tests.want.Items))
		}
		if got.Items[0].Name != tests.want.Items[0].Name {
			t.Errorf("getDeployment() returned wrong deployment got: %s, expected: %s", got.Items[0].Name, tests.want.Items[0].Name)
		}
		if got.Items[0].Namespace != tests.want.Items[0].Namespace {
			t.Errorf("getDeployment() returned wrong deployment got: %s, expected: %s", got.Items[0].Namespace, tests.want.Items[0].Namespace)
		}
		if *got.Items[0].Spec.Replicas != *tests.want.Items[0].Spec.Replicas {
			t.Errorf("getDeployment() returned wrong replica count got: %v, expected: %v", *got.Items[0].Spec.Replicas, *tests.want.Items[0].Spec.Replicas)
		}
	})
}

func TestGetDeploymentNegative(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	type args struct {
		namespace string
		clientset kubernetes.Interface
	}
	tests := []struct {
		name string
		args args
		want *appsv1.DeploymentList
	}{
		{
			name: "Test with invalid namespace",
			args: args{
				namespace: "testns",
				clientset: clientset,
			},
			want: &appsv1.DeploymentList{},
		},
		{
			name: "Test with invalid clientset",
			args: args{
				namespace: "testns1",
				clientset: clientset,
			},
			want: &appsv1.DeploymentList{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch name := tt.name; name {
			case "Test with invalid namespace":
				got, err := getDeployment(tt.args.namespace, tt.args.clientset)
				if err != nil {
					t.Errorf("%v", err)
				}
				if len(got.Items) != 0 {
					t.Errorf("getDeployment() returned wrong deployment for non-existent namespace got: %d, expected: %d\n", len(got.Items), len(tt.want.Items))
				}
				// case "Test with invalid clientset":
				// 	if err != nil {
				// 		t.Errorf("failed to delete collection")
				// 	}
				// 	_, err := getDeployment(tt.args.namespace, tt.args.clientset)
				// 	if err == nil {
				// 		t.Errorf("getDeployment() returned no error. expected error to be returned")
				// 	}
			}
		})

	}

}

func TestStoreDeploymentByNamespacePositive(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	type args struct {
		namespaces []string
		clientset  kubernetes.Interface
	}
	expectedTestDeployment1List := []appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "testns1",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
			},
		},
	}
	expectedTestDeployment2List := []appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "testns2",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
			},
		},
	}
	expectedMap := make(map[string][]appsv1.Deployment)
	expectedMap["testns1"] = expectedTestDeployment1List
	expectedMap["testns2"] = expectedTestDeployment2List
	tests := struct {
		name string
		args args
		want map[string][]appsv1.Deployment
	}{
		name: "Test with valid params",
		args: args{
			namespaces: []string{"testns1", "testns2"},
			clientset:  clientset,
		},
		want: expectedMap,
	}
	t.Run(tests.name, func(t *testing.T) {
		got, err := storeDeploymentsByNamespace(tests.args.namespaces, tests.args.clientset)
		if err != nil {
			t.Errorf("storeDeploymentsByNamespace() returned error: %v\n", err)
		}
		if len(got) != len(tests.want) {
			t.Fatalf("storeDeploymentsByNamespace() returned empty map got: %d, expected: %d", len(got), len(tests.want))
		}
		if got["testns1"][0].Name != tests.want["testns1"][0].Name {
			t.Errorf("storeDeploymentsByNamespace() returned wrong deployment name got: %s, expected: %s", got["testns1"][0].Name, tests.want["testns1"][0].Name)
		}
		if got["testns1"][0].Namespace != tests.want["testns1"][0].Namespace {
			t.Errorf("storeDeploymentsByNamespace() returned wrong deployment namespace got: %s, expected: %s", got["testns1"][0].Namespace, tests.want["testns1"][0].Namespace)
		}
		if *got["testns1"][0].Spec.Replicas != *tests.want["testns1"][0].Spec.Replicas {
			t.Errorf("storeDeploymentsByNamespace() returned wrong deployment replica count got: %d, expected: %d", got["testns1"][0].Spec.Replicas, tests.want["testns1"][0].Spec.Replicas)
		}
		if got["testns2"][0].Name != tests.want["testns2"][0].Name {
			t.Errorf("storeDeploymentsByNamespace() returned wrong deployment name got: %s, expected: %s", got["testns2"][0].Name, tests.want["testns2"][0].Name)
		}
		if got["testns2"][0].Namespace != tests.want["testns2"][0].Namespace {
			t.Errorf("storeDeploymentsByNamespace() returned wrong deployment namespace got: %s, expected: %s", got["testns2"][0].Namespace, tests.want["testns2"][0].Namespace)
		}
		if *got["testns2"][0].Spec.Replicas != *tests.want["testns2"][0].Spec.Replicas {
			t.Errorf("storeDeploymentsByNamespace() returned wrong deployment replica count got: %d, expected: %d", got["testns2"][0].Spec.Replicas, tests.want["testns2"][0].Spec.Replicas)
		}
	})

}

func TestStoreDeploymentByNamespaceNegative(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	type args struct {
		namespaces []string
		clientset  kubernetes.Interface
	}
	expectedTestDeployment1List := []appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment-1",
				Namespace: "testns1",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
			},
		},
	}
	expectedTestDeployment2List := []appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment-2",
				Namespace: "testns2",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
			},
		},
	}
	expectedMap := make(map[string][]appsv1.Deployment)
	expectedMap["testns1"] = expectedTestDeployment1List
	expectedMap["testns2"] = expectedTestDeployment2List
	tests := []struct {
		name string
		args args
		want map[string][]appsv1.Deployment
	}{
		{
			name: "Test with one no deployment namespace",
			args: args{
				namespaces: []string{"testns1", "testns2", "testns3"},
				clientset:  clientset,
			},
			want: expectedMap,
		},
		{
			name: "Test with invalid namespace",
			args: args{
				namespaces: []string{""},
				clientset:  clientset,
			},
			want: make(map[string][]appsv1.Deployment),
		},
		{
			name: "Test with empty namespace list",
			args: args{
				namespaces: []string{},
				clientset:  clientset,
			},
			want: make(map[string][]appsv1.Deployment),
		},
		{
			name: "Test with no deployment in namespace",
			args: args{
				namespaces: []string{"testns3"},
				clientset:  clientset,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch name := tt.name; name {
			case "Test with one no deployment namespace":
				got, err := storeDeploymentsByNamespace(tt.args.namespaces, tt.args.clientset)
				if err != nil {
					t.Errorf("storeDeploymentsByNamespace() returned error: %v\n", err)
				}
				if len(got) > len(tt.want) {
					t.Errorf("storeDeploymentsByNamespace() returned empty map got: %d, expected: %d", len(got), len(tt.want))
				}

			case "Test with invalid namespace":
				got, err := storeDeploymentsByNamespace(tt.args.namespaces, tt.args.clientset)
				if err == nil {
					t.Errorf("storeDeploymentsByNamespace() returned no error. expected error to be returned")
				}
				if len(got) != 0 {
					t.Errorf("storeDeploymentsByNamespace() returned non empty map. got: %d, expected: %d", len(got), len(tt.want))
				}
			case "Test with empty namespace list":
				got, err := storeDeploymentsByNamespace(tt.args.namespaces, tt.args.clientset)
				if err == nil {
					t.Errorf("storeDeploymentsByNamespace() returned no error. expected error to be returned")
				}
				if len(got) != 0 {
					t.Errorf("storeDeploymentsByNamespace() returned non empty map. got: %d, expected: %d", len(got), len(tt.want))
				}
			case "Test with no deployment in namespace":
				got, err := storeDeploymentsByNamespace(tt.args.namespaces, tt.args.clientset)
				if err == nil {
					t.Errorf("storeDeploymentByNamespace returned no error. expected error to be returned. got: %v", got)
				}
			}
		})
	}

}
func TestCheckDeploymentStatusPostive(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	testDeployment := appsv1.Deployment{
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
							Image: "nginx:latest",
						},
					},
				},
			},
		},
	}
	type args struct {
		namespace  string
		deployment appsv1.Deployment
		clientset  kubernetes.Interface
	}
	tests := struct {
		name string
		args args
	}{
		name: "Test with valid parms",
		args: args{
			namespace:  "testns1",
			deployment: testDeployment,
			clientset:  clientset,
		},
	}
	t.Run(tests.name, func(t *testing.T) {
		err := checkDeploymentStatus(tests.args.namespace, tests.args.deployment, tests.args.clientset)
		if err != nil {
			t.Errorf("checkDeploymentStatus() should not return an error, got: %v", err)
		}
	})

}

func TestCheckDeploymentStatusNegative(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	//defer TeardownTestEnvironment(t, clientset)
	testReplica := int32(0)
	testDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "testns1",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &testReplica,
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
							Image: "nginx:latest",
						},
					},
				},
			},
		},
	}
	type args struct {
		namespace  string
		deployment appsv1.Deployment
		clientset  kubernetes.Interface
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test with invalid expectation",
			args: args{
				namespace:  "testns1",
				deployment: testDeployment,
				clientset:  clientset,
			},
		},
		{
			name: "Test with faulty deployment",
			args: args{
				namespace:  "testns3",
				deployment: helper.TestFaultyDeployment,
				clientset:  clientset,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch name := tt.name; name {
			case "Test with invalid expectation":
				err := checkDeploymentStatus(tt.args.namespace, tt.args.deployment, tt.args.clientset)
				if err == nil {
					t.Errorf("checkDeploymentStatus() should return an error, got: %v", err)
				}
			case "Test with faulty deployment":
				_, err := clientset.AppsV1().Deployments(helper.TestFaultyDeployment.Namespace).Create(context.Background(), &helper.TestFaultyDeployment, metav1.CreateOptions{})
				if err != nil {
					t.Errorf("failed to create faulty deployment %v, reason: %v", helper.TestFaultyDeployment.Name, err)
				}
				time.Sleep(15 * time.Second)
				err = checkDeploymentStatus(tt.args.namespace, tt.args.deployment, tt.args.clientset)
				if err == nil {
					t.Errorf("checkDeploymentStatus() should return an error, got %v", err)
				}
			}
		})
	}

}

func TestValidateDeploymentsByNamespacePositive(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	type args struct {
		namespaces             []string
		deploymentsByNamespace map[string][]appsv1.Deployment
		clientset              kubernetes.Interface
		interval               time.Duration
		timeout                time.Duration
	}
	actualMap := make(map[string][]appsv1.Deployment)
	actualMap["testns1"] = []appsv1.Deployment{helper.TestDeployment1}
	actualMap["testns2"] = []appsv1.Deployment{helper.TestDeployment2}
	tests := struct {
		name string
		args args
	}{
		name: "Test with valid params",
		args: args{
			namespaces:             []string{"testns1", "testns2"},
			deploymentsByNamespace: actualMap,
			clientset:              clientset,
			interval:               time.Second,
			timeout:                time.Second,
		},
	}
	t.Run(tests.name, func(t *testing.T) {
		err := validateDeploymentsByNamespace(tests.args.namespaces, tests.args.deploymentsByNamespace, tests.args.clientset, tests.args.interval, tests.args.timeout)
		if err != nil {
			t.Errorf("validateDeploymentsByNamespace() should not return an error. got: %v", err)
		}
	})
}

func TestValidateDeploymentsByNamespaceNegative(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	//defer TeardownTestEnvironment(t,clientset)
	type args struct {
		namespaces             []string
		deploymentsByNamespace map[string][]appsv1.Deployment
		clientset              kubernetes.Interface
		interval               time.Duration
		timeout                time.Duration
	}
	testFaultyDeployment := []appsv1.Deployment{
		{
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
								Image:   "nginx:latest",
								Command: []string{"docker ps -a"},
							},
						},
					},
				},
			},
		},
	}
	actualMap := make(map[string][]appsv1.Deployment)
	actualMap["testns3"] = testFaultyDeployment
	actualMap["testns1"] = []appsv1.Deployment{helper.TestDeployment1}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test with empty namespace list",
			args: args{
				namespaces:             []string{},
				deploymentsByNamespace: make(map[string][]appsv1.Deployment),
				clientset:              clientset,
				interval:               1 * time.Second,
				timeout:                1 * time.Second,
			},
		},
		{
			name: "Test with empty deployments",
			args: args{
				namespaces:             []string{"testns1"},
				deploymentsByNamespace: make(map[string][]appsv1.Deployment),
				clientset:              clientset,
				interval:               1 * time.Second,
				timeout:                1 * time.Second,
			},
		},
		{
			name: "Test with invalid deployment status",
			args: args{
				namespaces:             []string{"testns3"},
				deploymentsByNamespace: actualMap,
				clientset:              clientset,
				interval:               5 * time.Second,
				timeout:                5 * time.Second,
			},
		},
		{
			name: "Test with invalid timeout/interval duration",
			args: args{
				namespaces:             []string{"testns1"},
				deploymentsByNamespace: actualMap,
				clientset:              clientset,
				interval:               -1 * time.Minute,
				timeout:                0 * time.Minute,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch name := tt.name; name {
			case "Test with empty namespace list":
				err := validateDeploymentsByNamespace(tt.args.namespaces, tt.args.deploymentsByNamespace, tt.args.clientset, tt.args.interval, tt.args.timeout)
				if err == nil {
					t.Errorf("validateDeploymentsByNamespace() should return an error. got: %v", err)
				}
			case "Test with empty deployments":
				err := validateDeploymentsByNamespace(tt.args.namespaces, tt.args.deploymentsByNamespace, tt.args.clientset, tt.args.interval, tt.args.timeout)
				if err == nil {
					t.Errorf("validateDeploymentsByNamespace() should return an error. got: %v", err)
				}
			case "Test with invalid deployment status":
				err := validateDeploymentsByNamespace(tt.args.namespaces, tt.args.deploymentsByNamespace, tt.args.clientset, tt.args.interval, tt.args.timeout)
				if err == nil {
					t.Errorf("validateDeploymentsByNamespace() should return an error. got: %v", err)
				}
			case "Test with invalid timeout/interval duration":
				err := validateDeploymentsByNamespace(tt.args.namespaces, tt.args.deploymentsByNamespace, tt.args.clientset, tt.args.interval, tt.args.timeout)
				if err == nil {
					t.Errorf("validateDeploymentsByNamespace() should return an error. got: %v", err)
				}
			}
		})
	}
}

func TestCheckDeploymentsPositive(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	type args struct {
		namespace []string
		clientset kubernetes.Interface
		interval  time.Duration
		timeout   time.Duration
	}
	tests := struct {
		name string
		args args
	}{
		name: "Test with valid params",
		args: args{
			namespace: []string{"testns1", "testns2"},
			clientset: clientset,
			interval:  time.Second,
			timeout:   time.Second,
		},
	}
	t.Run(tests.name, func(t *testing.T) {
		err := CheckDeployments(tests.args.namespace, tests.args.clientset, tests.args.timeout, tests.args.interval)
		if err != nil {
			t.Errorf("CheckDeployments() should not return an error. got: %v", err)
		}
	})
}

func TestCheckDeploymentsNegative(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	type args struct {
		namespace []string
		clientset kubernetes.Interface
		interval  time.Duration
		timeout   time.Duration
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test with empty namespace list",
			args: args{
				namespace: []string{},
				clientset: clientset,
				interval:  time.Second,
				timeout:   time.Second,
			},
		},
		{
			name: "Test with faulty deployment",
			args: args{
				namespace: []string{"testns3"},
				clientset: clientset,
				interval:  time.Second,
				timeout:   time.Second,
			},
		},
		{
			name: "Test with invalid namespace list",
			args: args{
				namespace: []string{"faultyns"},
				clientset: clientset,
				interval:  time.Second,
				timeout:   time.Second,
			},
		},
		{
			name: "Test with invalid timeout/interval",
			args: args{
				namespace: []string{"testns1"},
				clientset: clientset,
				interval:  -1 * time.Minute,
				timeout:   0 * time.Second,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckDeployments(tt.args.namespace, tt.args.clientset, tt.args.timeout, tt.args.interval)
			if err == nil {
				t.Errorf("CheckDeployments() should return an error. got: %v", err)
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
