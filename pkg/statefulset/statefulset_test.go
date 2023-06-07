package statefulset

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
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

func TestGetStatefulSetPositive(t *testing.T) {
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
		want *appsv1.StatefulSetList
	}{
		name: "Test with valid params",
		args: args{
			namespace: "testns1",
			clientset: clientset,
		},
		want: &appsv1.StatefulSetList{
			Items: []appsv1.StatefulSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-statefulset",
						Namespace: "testns1",
					},
					Spec: appsv1.StatefulSetSpec{
						Replicas: &replicas,
					},
				},
			},
		},
	}
	t.Run(tests.name, func(t *testing.T) {
		got, err := getStatefulSet(tests.args.namespace, tests.args.clientset)
		if err != nil {
			t.Errorf("%v", err)
		}
		if len(got.Items) != 1 {
			t.Fatalf("getStatefulSet() returned wrong number of statefulsets: %d, expected: %d\n", len(got.Items), len(tests.want.Items))
		}
		if got.Items[0].Name != tests.want.Items[0].Name {
			t.Errorf("getStatefulSet() returned wrong statefulset got: %s, expected: %s", got.Items[0].Name, tests.want.Items[0].Name)
		}
		if got.Items[0].Namespace != tests.want.Items[0].Namespace {
			t.Errorf("getStatefulSet() returned wrong statefulset got: %s, expected: %s", got.Items[0].Namespace, tests.want.Items[0].Namespace)
		}
		if *got.Items[0].Spec.Replicas != *tests.want.Items[0].Spec.Replicas {
			t.Errorf("getStatefulSet() returned wrong replica count got: %v, expected: %v", *got.Items[0].Spec.Replicas, *tests.want.Items[0].Spec.Replicas)
		}
	})
}

func TestGetStatefulSetNegative(t *testing.T) {
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
		want *appsv1.StatefulSetList
	}{
		{
			name: "Test with invalid namespace",
			args: args{
				namespace: "testns",
				clientset: clientset,
			},
			want: &appsv1.StatefulSetList{},
		},
		{
			name: "Test with invalid clientset",
			args: args{
				namespace: "testns1",
				clientset: clientset,
			},
			want: &appsv1.StatefulSetList{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch name := tt.name; name {
			case "Test with invalid namespace":
				got, err := getStatefulSet(tt.args.namespace, tt.args.clientset)
				if err != nil {
					t.Errorf("%v", err)
				}
				if len(got.Items) != 0 {
					t.Errorf("getStatefulSet() returned wrong statefulset for non-existent namespace got: %d, expected: %d\n", len(got.Items), len(tt.want.Items))
				}
				// case "Test with invalid clientset":
				// 	if err != nil {
				// 		t.Errorf("failed to delete collection")
				// 	}
				// 	_, err := getStatefulSet(tt.args.namespace, tt.args.clientset)
				// 	if err == nil {
				// 		t.Errorf("getStatefulSet() returned no error. expected error to be returned")
				// 	}
			}
		})

	}

}

func TestStoreStatefulSetByNamespacePositive(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	type args struct {
		namespaces []string
		clientset  kubernetes.Interface
	}
	expectedTestStatefulSet1List := []appsv1.StatefulSet{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "testns1",
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: &replicas,
			},
		},
	}
	expectedTestStatefulSet2List := []appsv1.StatefulSet{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "testns2",
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: &replicas,
			},
		},
	}
	expectedMap := make(map[string][]appsv1.StatefulSet)
	expectedMap["testns1"] = expectedTestStatefulSet1List
	expectedMap["testns2"] = expectedTestStatefulSet2List
	tests := struct {
		name string
		args args
		want map[string][]appsv1.StatefulSet
	}{
		name: "Test with valid params",
		args: args{
			namespaces: []string{"testns1", "testns2"},
			clientset:  clientset,
		},
		want: expectedMap,
	}
	t.Run(tests.name, func(t *testing.T) {
		got, err := storeStatefulSetsByNamespace(tests.args.namespaces, tests.args.clientset)
		if err != nil {
			t.Errorf("storeStatefulSetsByNamespace() returned error: %v\n", err)
		}
		if len(got) != len(tests.want) {
			t.Fatalf("storeStatefulSetsByNamespace() returned empty map got: %d, expected: %d", len(got), len(tests.want))
		}
		if got["testns1"][0].Name != tests.want["testns1"][0].Name {
			t.Errorf("storeStatefulSetsByNamespace() returned wrong statefulset name got: %s, expected: %s", got["testns1"][0].Name, tests.want["testns1"][0].Name)
		}
		if got["testns1"][0].Namespace != tests.want["testns1"][0].Namespace {
			t.Errorf("storeStatefulSetsByNamespace() returned wrong statefulset namespace got: %s, expected: %s", got["testns1"][0].Namespace, tests.want["testns1"][0].Namespace)
		}
		if *got["testns1"][0].Spec.Replicas != *tests.want["testns1"][0].Spec.Replicas {
			t.Errorf("storeStatefulSetsByNamespace() returned wrong statefulset replica count got: %d, expected: %d", got["testns1"][0].Spec.Replicas, tests.want["testns1"][0].Spec.Replicas)
		}
		if got["testns2"][0].Name != tests.want["testns2"][0].Name {
			t.Errorf("storeStatefulSetsByNamespace() returned wrong statefulset name got: %s, expected: %s", got["testns2"][0].Name, tests.want["testns2"][0].Name)
		}
		if got["testns2"][0].Namespace != tests.want["testns2"][0].Namespace {
			t.Errorf("storeStatefulSetsByNamespace() returned wrong statefulset namespace got: %s, expected: %s", got["testns2"][0].Namespace, tests.want["testns2"][0].Namespace)
		}
		if *got["testns2"][0].Spec.Replicas != *tests.want["testns2"][0].Spec.Replicas {
			t.Errorf("storeStatefulSetsByNamespace() returned wrong statefulset replica count got: %d, expected: %d", got["testns2"][0].Spec.Replicas, tests.want["testns2"][0].Spec.Replicas)
		}
	})

}

func TestStoreStatefulSetByNamespaceNegative(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	type args struct {
		namespaces []string
		clientset  kubernetes.Interface
	}
	expectedTestStatefulSet1List := []appsv1.StatefulSet{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset-1",
				Namespace: "testns1",
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: &replicas,
			},
		},
	}
	expectedTestStatefulSet2List := []appsv1.StatefulSet{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset-2",
				Namespace: "testns2",
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: &replicas,
			},
		},
	}
	expectedMap := make(map[string][]appsv1.StatefulSet)
	expectedMap["testns1"] = expectedTestStatefulSet1List
	expectedMap["testns2"] = expectedTestStatefulSet2List
	tests := []struct {
		name string
		args args
		want map[string][]appsv1.StatefulSet
	}{
		{
			name: "Test with one no statefulset namespace",
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
			want: make(map[string][]appsv1.StatefulSet),
		},
		{
			name: "Test with empty namespace list",
			args: args{
				namespaces: []string{},
				clientset:  clientset,
			},
			want: make(map[string][]appsv1.StatefulSet),
		},
		{
			name: "Test with no statefulset in namespace",
			args: args{
				namespaces: []string{"testns3"},
				clientset:  clientset,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch name := tt.name; name {
			case "Test with one no statefulset namespace":
				got, err := storeStatefulSetsByNamespace(tt.args.namespaces, tt.args.clientset)
				if err != nil {
					t.Errorf("storeStatefulSetsByNamespace() returned error: %v\n", err)
				}
				if len(got) > len(tt.want) {
					t.Errorf("storeStatefulSetsByNamespace() returned empty map got: %d, expected: %d", len(got), len(tt.want))
				}

			case "Test with invalid namespace":
				got, err := storeStatefulSetsByNamespace(tt.args.namespaces, tt.args.clientset)
				if err == nil {
					t.Errorf("storeStatefulSetsByNamespace() returned no error. expected error to be returned")
				}
				if len(got) != 0 {
					t.Errorf("storeStatefulSetsByNamespace() returned non empty map. got: %d, expected: %d", len(got), len(tt.want))
				}
			case "Test with empty namespace list":
				got, err := storeStatefulSetsByNamespace(tt.args.namespaces, tt.args.clientset)
				if err == nil {
					t.Errorf("storeStatefulSetsByNamespace() returned no error. expected error to be returned")
				}
				if len(got) != 0 {
					t.Errorf("storeStatefulSetsByNamespace() returned non empty map. got: %d, expected: %d", len(got), len(tt.want))
				}
			case "Test with no statefulset in namespace":
				got, err := storeStatefulSetsByNamespace(tt.args.namespaces, tt.args.clientset)
				if err == nil {
					t.Errorf("storeStatefulSetsByNamespace returned no error. expected error to be returned. got: %v", got)
				}
			}
		})
	}

}
func TestCheckStatefulSetStatusPostive(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	testStatefulSet := appsv1.StatefulSet{
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
							Image: "nginx:latest",
						},
					},
				},
			},
		},
	}
	type args struct {
		namespace   string
		statefulset appsv1.StatefulSet
		clientset   kubernetes.Interface
	}
	tests := struct {
		name string
		args args
	}{
		name: "Test with valid parms",
		args: args{
			namespace:   "testns1",
			statefulset: testStatefulSet,
			clientset:   clientset,
		},
	}
	t.Run(tests.name, func(t *testing.T) {
		err := checkStatefulSetStatus(tests.args.namespace, tests.args.statefulset, tests.args.clientset)
		if err != nil {
			t.Errorf("checkStatefulSetStatus() should not return an error, got: %v", err)
		}
	})

}

func TestCheckStatefulSetStatusNegative(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	testReplica := int32(0)
	testStatefulSet := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-statefulset",
			Namespace: "testns1",
		},
		Spec: appsv1.StatefulSetSpec{
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
	testFaultyStatefulSet := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-statefulset",
			Namespace: "testns3",
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
							Name:    "my-container",
							Image:   "nginx:latest",
							Command: []string{"docker ps -a"},
						},
					},
				},
			},
		},
	}
	type args struct {
		namespace   string
		statefulset appsv1.StatefulSet
		clientset   kubernetes.Interface
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test with invalid expectation",
			args: args{
				namespace:   "testns1",
				statefulset: testStatefulSet,
				clientset:   clientset,
			},
		},
		{
			name: "Test with faulty statefulset",
			args: args{
				namespace:   "testns3",
				statefulset: testFaultyStatefulSet,
				clientset:   clientset,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch name := tt.name; name {
			case "Test with invalid expectation":
				err := checkStatefulSetStatus(tt.args.namespace, tt.args.statefulset, tt.args.clientset)
				if err == nil {
					t.Errorf("checkStatefulSetStatus() should return an error, got: %v", err)
				}
			case "Test with faulty statefulset":
				_, err := clientset.AppsV1().StatefulSets(testFaultyStatefulSet.Namespace).Create(context.Background(), &testFaultyStatefulSet, metav1.CreateOptions{})
				if err != nil {
					t.Errorf("failed to create faulty statefulset %v, reason: %v", testStatefulSet.Name, err)
				}
				time.Sleep(15 * time.Second)
				err = checkStatefulSetStatus(tt.args.namespace, tt.args.statefulset, tt.args.clientset)
				if err == nil {
					t.Errorf("checkStatefulSetStatus() should return an error, got %v", err)
				}
			}
		})
	}

}

func TestValidateStatefulSetsByNamespacePositive(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	type args struct {
		namespaces              []string
		statefulSetsByNamespace map[string][]appsv1.StatefulSet
		clientset               kubernetes.Interface
		interval                time.Duration
		timeout                 time.Duration
	}
	actualMap := make(map[string][]appsv1.StatefulSet)
	actualMap["testns1"] = []appsv1.StatefulSet{helper.TestStatefulSet1}
	actualMap["testns2"] = []appsv1.StatefulSet{helper.TestStatefulSet2}
	tests := struct {
		name string
		args args
	}{
		name: "Test with valid params",
		args: args{
			namespaces:              []string{"testns1", "testns2"},
			statefulSetsByNamespace: actualMap,
			clientset:               clientset,
			interval:                time.Second,
			timeout:                 time.Second,
		},
	}
	t.Run(tests.name, func(t *testing.T) {
		err := validateStatefulSetsByNamespace(tests.args.namespaces, tests.args.statefulSetsByNamespace, tests.args.clientset, tests.args.interval, tests.args.timeout)
		if err != nil {
			t.Errorf("validateStatefulSetsByNamespace() should not return an error. got: %v", err)
		}
	})
}

func TestValidateStatefulSetsByNamespaceNegative(t *testing.T) {
	res := testing.Verbose()
	if !res {
		log.SetOutput(io.Discard)
	}
	//defer TeardownTestEnvironment(t,clientset)
	type args struct {
		namespaces              []string
		statefulSetsByNamespace map[string][]appsv1.StatefulSet
		clientset               kubernetes.Interface
		interval                time.Duration
		timeout                 time.Duration
	}
	testFaultyStatefulSet := []appsv1.StatefulSet{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "testns3",
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
	actualMap := make(map[string][]appsv1.StatefulSet)
	actualMap["testns3"] = testFaultyStatefulSet
	actualMap["testns1"] = []appsv1.StatefulSet{helper.TestStatefulSet1}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test with empty namespace list",
			args: args{
				namespaces:              []string{},
				statefulSetsByNamespace: make(map[string][]appsv1.StatefulSet),
				clientset:               clientset,
				interval:                1 * time.Second,
				timeout:                 1 * time.Second,
			},
		},
		{
			name: "Test with empty statefulsets",
			args: args{
				namespaces:              []string{"testns1"},
				statefulSetsByNamespace: make(map[string][]appsv1.StatefulSet),
				clientset:               clientset,
				interval:                1 * time.Second,
				timeout:                 1 * time.Second,
			},
		},
		{
			name: "Test with invalid statefulset status",
			args: args{
				namespaces:              []string{"testns3"},
				statefulSetsByNamespace: actualMap,
				clientset:               clientset,
				interval:                5 * time.Second,
				timeout:                 5 * time.Second,
			},
		},
		{
			name: "Test with invalid timeout/interval duration",
			args: args{
				namespaces:              []string{"testns1"},
				statefulSetsByNamespace: actualMap,
				clientset:               clientset,
				interval:                -1 * time.Minute,
				timeout:                 0 * time.Minute,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch name := tt.name; name {
			case "Test with empty namespace list":
				err := validateStatefulSetsByNamespace(tt.args.namespaces, tt.args.statefulSetsByNamespace, tt.args.clientset, tt.args.interval, tt.args.timeout)
				if err == nil {
					t.Errorf("validateStatefulSetsByNamespace() should return an error. got: %v", err)
				}
			case "Test with empty statefulsets":
				err := validateStatefulSetsByNamespace(tt.args.namespaces, tt.args.statefulSetsByNamespace, tt.args.clientset, tt.args.interval, tt.args.timeout)
				if err == nil {
					t.Errorf("validateStatefulSetsByNamespace() should return an error. got: %v", err)
				}
			case "Test with invalid statefulset status":
				err := validateStatefulSetsByNamespace(tt.args.namespaces, tt.args.statefulSetsByNamespace, tt.args.clientset, tt.args.interval, tt.args.timeout)
				if err == nil {
					t.Errorf("validateStatefulSetsByNamespace() should return an error. got: %v", err)
				}
			case "Test with invalid timeout/interval duration":
				err := validateStatefulSetsByNamespace(tt.args.namespaces, tt.args.statefulSetsByNamespace, tt.args.clientset, tt.args.interval, tt.args.timeout)
				if err == nil {
					t.Errorf("validateStatefulSetsByNamespace() should return an error. got: %v", err)
				}
			}
		})
	}
}

func TestCheckStatefulSetsPositive(t *testing.T) {
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
		err := CheckStatefulSets(tests.args.namespace, tests.args.clientset, tests.args.timeout, tests.args.interval)
		if err != nil {
			t.Errorf("CheckStatefulSets() should not return an error. got: %v", err)
		}
	})
}

func TestCheckStatefulSetsNegative(t *testing.T) {
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
			name: "Test with faulty statefulset",
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
			err := CheckStatefulSets(tt.args.namespace, tt.args.clientset, tt.args.timeout, tt.args.interval)
			if err == nil {
				t.Errorf("CheckStatefulSets() should return an error. got: %v", err)
			}
		})
	}
}

func TestMain(m *testing.M) {
	cls, err := helper.SetupTestEnvironment("statefulsets")
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
