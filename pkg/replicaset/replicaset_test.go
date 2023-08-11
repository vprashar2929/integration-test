package replicaset

import (
	"testing"
	"time"

	"github.com/vprashar2929/integration-test/pkg/logger"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	testNS             = "test-namespace"
	testReplicaSet     = "test-replicaset"
	testLabels         = make(map[string]string)
	testReplicaSetList = appsv1.ReplicaSetList{
		Items: []appsv1.ReplicaSet{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testReplicaSet,
					Namespace: testNS,
				},
				Spec: appsv1.ReplicaSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: testLabels,
					},
				},
			},
		},
	}
)

// Limitation: Since retry.RetryOnConflict only works on a running Kubernetes cluster.
// we use fake to build client we have to agument the way retryOnConflict works.
// TODO: Come up with good way to handle this

type mockRetryer struct {
	err error
}

func (m *mockRetryer) RetryOnConflict(backoff wait.Backoff, fn func() error) error {
	return m.err
}

func TestGetReplicaSet(t *testing.T) {
	clientset := fake.NewSimpleClientset(&testReplicaSetList)
	logger.NewLogger(logger.LevelInfo)
	rset, err := getReplicaSet(testNS, clientset)
	if err != nil {
		t.Fatalf("expected nil got: %v", err)
	}
	if len(rset.Items) != 1 {
		t.Errorf("expected 1 replicaset, got: %v", len(rset.Items))
	}
}

func TestGetReplicaSetNoReplicaSet(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	logger.NewLogger(logger.LevelInfo)
	_, err := getReplicaSet(testNS, clientset)
	if err != ErrNoReplicaSet {
		t.Fatalf("expected ErrNoReplicaSet, got: %v", err)
	}
}

func TestStoreReplicaSetsByNamespace(t *testing.T) {
	clientset := fake.NewSimpleClientset(&testReplicaSetList)
	namespaces := []string{testNS}
	replicasetsByNamespace, err := storeReplicaSetsByNamespace(namespaces, clientset)
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if len(replicasetsByNamespace) != 1 {
		t.Errorf("expected 1 replicaset, got: %v", len(replicasetsByNamespace))
	}
}

func TestStoreReplicaSetsByNamespaceNoNamespace(t *testing.T) {
	clientset := fake.NewSimpleClientset(&testReplicaSetList)
	logger.NewLogger(logger.LevelInfo)
	namespaces := []string{}
	_, err := storeReplicaSetsByNamespace(namespaces, clientset)
	if err != ErrNamespaceEmpty {
		t.Fatalf("expected ErrNamespaceEmpty, got: %v", err)
	}
}

func TestStoreReplicaSetsByNamespaceNoReplicaSets(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	logger.NewLogger(logger.LevelInfo)
	namespaces := []string{testNS}
	_, err := storeReplicaSetsByNamespace(namespaces, clientset)
	if err != ErrNoReplicaSet {
		t.Fatalf("expected ErrNoReplicaSet, got: %v", err)
	}
}

func TestCheckReplicaSetStatus(t *testing.T) {
	clientset := fake.NewSimpleClientset(&testReplicaSetList)
	// TODO: Come up with good way to write this
	retryer = &mockRetryer{err: nil}
	logger.NewLogger(logger.LevelInfo)
	err := checkReplicaSetsStatus(testNS, testReplicaSetList.Items[0], clientset)
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
}

func TestCheckReplicaSetStatusNotHealthy(t *testing.T) {
	faultyRS := appsv1.ReplicaSetList{
		Items: []appsv1.ReplicaSet{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testReplicaSet,
					Namespace: testNS,
				},
				Status: appsv1.ReplicaSetStatus{
					AvailableReplicas: 0,
				},
			},
		},
	}
	clientset := fake.NewSimpleClientset(&faultyRS)
	// TODO: Come up with good way to write this
	retryer = &mockRetryer{err: ErrReplicaSetNotHealthy}
	logger.NewLogger(logger.LevelInfo)
	err := checkReplicaSetsStatus(testNS, faultyRS.Items[0], clientset)
	if err != ErrReplicaSetNotHealthy {
		t.Fatalf("expected ErrReplicaSetNotHealthy, got: %v", err)
	}
}

func TestValidateReplicaSetsByNamespace(t *testing.T) {
	testLabels["app"] = "test-app"
	testPods := corev1.PodList{
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testPod",
					Namespace: testNS,
					Labels:    testLabels,
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
		},
	}
	clientset := fake.NewSimpleClientset(&testReplicaSetList, &testPods)
	// TODO: Come up with good way to write this
	retryer = &mockRetryer{err: nil}
	logger.NewLogger(logger.LevelInfo)
	namespaces := []string{testNS}
	replicasetsByNamespace := make(map[string][]appsv1.ReplicaSet)
	replicasetsByNamespace[testNS] = testReplicaSetList.Items
	interval := 1 * time.Second
	timeout := 5 * time.Second
	err := validateReplicaSetsByNamespace(namespaces, replicasetsByNamespace, clientset, interval, timeout)
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
}

func TestValidateReplicaSetsByNamespaceInvalidInterval(t *testing.T) {
	testLabels["app"] = "test-app"
	testPods := corev1.PodList{
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testPod",
					Namespace: testNS,
					Labels:    testLabels,
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
		},
	}
	clientset := fake.NewSimpleClientset(&testReplicaSetList, &testPods)
	// TODO: Come up with good way to write this
	retryer = &mockRetryer{err: nil}
	logger.NewLogger(logger.LevelInfo)
	namespaces := []string{testNS}
	replicasetsByNamespace := make(map[string][]appsv1.ReplicaSet)
	replicasetsByNamespace[testNS] = testReplicaSetList.Items
	interval := -1 * time.Second
	timeout := -5 * time.Second
	err := validateReplicaSetsByNamespace(namespaces, replicasetsByNamespace, clientset, interval, timeout)
	if err != ErrInvalidInterval {
		t.Fatalf("expected ErrInvalidInterval, got: %v", err)
	}
}

func TestValidateReplicaSetsByNamespaceReplicaSetFailed(t *testing.T) {
	testLabels["app"] = "test-app"
	faultyPods := corev1.PodList{
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testPod",
					Namespace: testNS,
					Labels:    testLabels,
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodUnknown,
				},
			},
		},
	}
	faultyRS := appsv1.ReplicaSetList{
		Items: []appsv1.ReplicaSet{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testReplicaSet,
					Namespace: testNS,
				},
				Spec: appsv1.ReplicaSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: testLabels,
					},
				},
				Status: appsv1.ReplicaSetStatus{
					AvailableReplicas: 0,
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(&faultyRS, &faultyPods)
	// TODO: Come up with good way to write this
	retryer = &mockRetryer{err: ErrReplicaSetNotHealthy}
	logger.NewLogger(logger.LevelInfo)
	namespaces := []string{testNS}
	replicasetsByNamespace := make(map[string][]appsv1.ReplicaSet)
	replicasetsByNamespace[testNS] = faultyRS.Items
	interval := 1 * time.Second
	timeout := 5 * time.Second
	err := validateReplicaSetsByNamespace(namespaces, replicasetsByNamespace, clientset, interval, timeout)
	if err == nil {
		t.Fatalf("expected error, got: %v", err)
	}
}

func TestValidateReplicaSetsByNamespacePodFailed(t *testing.T) {
	testLabels["app"] = "test-app"
	faultyPods := corev1.PodList{
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testPod",
					Namespace: testNS,
					Labels:    testLabels,
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodUnknown,
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(&testReplicaSetList, &faultyPods)
	// TODO: Come up with good way to write this
	retryer = &mockRetryer{err: nil}
	logger.NewLogger(logger.LevelInfo)
	namespaces := []string{testNS}
	replicasetsByNamespace := make(map[string][]appsv1.ReplicaSet)
	replicasetsByNamespace[testNS] = testReplicaSetList.Items
	interval := 1 * time.Second
	timeout := 5 * time.Second
	err := validateReplicaSetsByNamespace(namespaces, replicasetsByNamespace, clientset, interval, timeout)
	if err == nil {
		t.Fatalf("expected error, got: %v", err)
	}
}

func TestCheckReplicaSets(t *testing.T) {
	testLabels["app"] = "test-app"
	testPods := corev1.PodList{
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testPod",
					Namespace: testNS,
					Labels:    testLabels,
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
		},
	}
	clientset := fake.NewSimpleClientset(&testReplicaSetList, &testPods)
	// TODO: Come up with good way to write this
	retryer = &mockRetryer{err: nil}
	logger.NewLogger(logger.LevelInfo)
	namespaces := []string{testNS}
	interval := 1 * time.Second
	timeout := 5 * time.Second
	err := CheckReplicaSets(namespaces, clientset, interval, timeout)
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
}

func TestCheckReplicaSetsNoNamespace(t *testing.T) {
	clientset := fake.NewSimpleClientset(&testReplicaSetList)
	// TODO: Come up with good way to write this
	retryer = &mockRetryer{err: nil}
	logger.NewLogger(logger.LevelInfo)
	namespaces := []string{}
	interval := 1 * time.Second
	timeout := 5 * time.Second
	err := CheckReplicaSets(namespaces, clientset, interval, timeout)
	if err != ErrNamespaceEmpty {
		t.Fatalf("expected ErrNamespaceEmpty, got: %v", err)
	}
}
