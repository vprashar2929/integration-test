package daemonset

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
	testNS            = "test-namespace"
	testDaemonSet     = "test-daemonset"
	testLabels        = make(map[string]string)
	testDaemonSetList = appsv1.DaemonSetList{
		Items: []appsv1.DaemonSet{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testDaemonSet,
					Namespace: testNS,
				},
				Spec: appsv1.DaemonSetSpec{
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

func TestGetDaemonSet(t *testing.T) {
	clientset := fake.NewSimpleClientset(&testDaemonSetList)
	logger.NewLogger(logger.LevelInfo)
	rset, err := getDaemonSet(testNS, clientset)
	if err != nil {
		t.Fatalf("expected nil got: %v", err)
	}
	if len(rset.Items) != 1 {
		t.Errorf("expected 1 daemonset, got: %v", len(rset.Items))
	}
}

func TestGetDaemonSetNoDaemonSet(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	logger.NewLogger(logger.LevelInfo)
	_, err := getDaemonSet(testNS, clientset)
	if err != ErrNoDaemonSet {
		t.Fatalf("expected ErrNoDaemonSet, got: %v", err)
	}
}

func TestStoreDaemonSetsByNamespace(t *testing.T) {
	clientset := fake.NewSimpleClientset(&testDaemonSetList)
	namespaces := []string{testNS}
	daemonsetsByNamespace, err := storeDaemonSetsByNamespace(namespaces, clientset)
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	if len(daemonsetsByNamespace) != 1 {
		t.Errorf("expected 1 daemonset, got: %v", len(daemonsetsByNamespace))
	}
}

func TestStoreDaemonSetsByNamespaceNoNamespace(t *testing.T) {
	clientset := fake.NewSimpleClientset(&testDaemonSetList)
	logger.NewLogger(logger.LevelInfo)
	namespaces := []string{}
	_, err := storeDaemonSetsByNamespace(namespaces, clientset)
	if err != ErrNamespaceEmpty {
		t.Fatalf("expected ErrNamespaceEmpty, got: %v", err)
	}
}

func TestStoreDaemonSetsByNamespaceNoDaemonSets(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	logger.NewLogger(logger.LevelInfo)
	namespaces := []string{testNS}
	_, err := storeDaemonSetsByNamespace(namespaces, clientset)
	if err != ErrNoDaemonSet {
		t.Fatalf("expected ErrNoDaemonSet, got: %v", err)
	}
}

func TestCheckDaemonSetStatus(t *testing.T) {
	clientset := fake.NewSimpleClientset(&testDaemonSetList)
	// TODO: Come up with good way to write this
	retryer = &mockRetryer{err: nil}
	logger.NewLogger(logger.LevelInfo)
	err := checkDaemonSetsStatus(testNS, testDaemonSetList.Items[0], clientset)
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
}

func TestCheckDaemonSetStatusNotHealthy(t *testing.T) {
	faultyDS := appsv1.DaemonSetList{
		Items: []appsv1.DaemonSet{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testDaemonSet,
					Namespace: testNS,
				},
				Status: appsv1.DaemonSetStatus{
					NumberUnavailable: 1,
				},
			},
		},
	}
	clientset := fake.NewSimpleClientset(&faultyDS)
	// TODO: Come up with good way to write this
	retryer = &mockRetryer{err: ErrDaemonSetNotHealthy}
	logger.NewLogger(logger.LevelInfo)
	err := checkDaemonSetsStatus(testNS, faultyDS.Items[0], clientset)
	if err != ErrDaemonSetNotHealthy {
		t.Fatalf("expected ErrDaemonSetNotHealthy, got: %v", err)
	}
}

func TestValidateDaemonSetsByNamespace(t *testing.T) {
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
	clientset := fake.NewSimpleClientset(&testDaemonSetList, &testPods)
	// TODO: Come up with good way to write this
	retryer = &mockRetryer{err: nil}
	logger.NewLogger(logger.LevelInfo)
	namespaces := []string{testNS}
	daemonsetsByNamespace := make(map[string][]appsv1.DaemonSet)
	daemonsetsByNamespace[testNS] = testDaemonSetList.Items
	interval := 1 * time.Second
	timeout := 5 * time.Second
	err := validateDaemonSetsByNamespace(namespaces, daemonsetsByNamespace, clientset, interval, timeout)
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
}

func TestValidateDaemonSetsByNamespaceInvalidInterval(t *testing.T) {
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
	clientset := fake.NewSimpleClientset(&testDaemonSetList, &testPods)
	// TODO: Come up with good way to write this
	retryer = &mockRetryer{err: nil}
	logger.NewLogger(logger.LevelInfo)
	namespaces := []string{testNS}
	daemonsetsByNamespace := make(map[string][]appsv1.DaemonSet)
	daemonsetsByNamespace[testNS] = testDaemonSetList.Items
	interval := -1 * time.Second
	timeout := -5 * time.Second
	err := validateDaemonSetsByNamespace(namespaces, daemonsetsByNamespace, clientset, interval, timeout)
	if err != ErrInvalidInterval {
		t.Fatalf("expected ErrInvalidInterval, got: %v", err)
	}
}

func TestValidateDaemonSetsByNamespaceDaemonSetFailed(t *testing.T) {
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
	faultyDS := appsv1.DaemonSetList{
		Items: []appsv1.DaemonSet{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testDaemonSet,
					Namespace: testNS,
				},
				Spec: appsv1.DaemonSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: testLabels,
					},
				},
				Status: appsv1.DaemonSetStatus{
					NumberUnavailable: 1,
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(&faultyDS, &faultyPods)
	// TODO: Come up with good way to write this
	retryer = &mockRetryer{err: ErrDaemonSetNotHealthy}
	logger.NewLogger(logger.LevelInfo)
	namespaces := []string{testNS}
	daemonsetsByNamespace := make(map[string][]appsv1.DaemonSet)
	daemonsetsByNamespace[testNS] = faultyDS.Items
	interval := 1 * time.Second
	timeout := 5 * time.Second
	err := validateDaemonSetsByNamespace(namespaces, daemonsetsByNamespace, clientset, interval, timeout)
	if err == nil {
		t.Fatalf("expected error, got: %v", err)
	}
}

func TestValidateDaemonSetsByNamespacePodFailed(t *testing.T) {
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

	clientset := fake.NewSimpleClientset(&testDaemonSetList, &faultyPods)
	// TODO: Come up with good way to write this
	retryer = &mockRetryer{err: nil}
	logger.NewLogger(logger.LevelInfo)
	namespaces := []string{testNS}
	daemonsetsByNamespace := make(map[string][]appsv1.DaemonSet)
	daemonsetsByNamespace[testNS] = testDaemonSetList.Items
	interval := 1 * time.Second
	timeout := 5 * time.Second
	err := validateDaemonSetsByNamespace(namespaces, daemonsetsByNamespace, clientset, interval, timeout)
	if err == nil {
		t.Fatalf("expected error, got: %v", err)
	}
}

func TestCheckDaemonSets(t *testing.T) {
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
	clientset := fake.NewSimpleClientset(&testDaemonSetList, &testPods)
	// TODO: Come up with good way to write this
	retryer = &mockRetryer{err: nil}
	logger.NewLogger(logger.LevelInfo)
	namespaces := []string{testNS}
	interval := 1 * time.Second
	timeout := 5 * time.Second
	err := CheckDaemonSets(namespaces, clientset, interval, timeout)
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
}

func TestCheckDaemonSetsNoNamespace(t *testing.T) {
	clientset := fake.NewSimpleClientset(&testDaemonSetList)
	// TODO: Come up with good way to write this
	retryer = &mockRetryer{err: nil}
	logger.NewLogger(logger.LevelInfo)
	namespaces := []string{}
	interval := 1 * time.Second
	timeout := 5 * time.Second
	err := CheckDaemonSets(namespaces, clientset, interval, timeout)
	if err != ErrNamespaceEmpty {
		t.Fatalf("expected ErrNamespaceEmpty, got: %v", err)
	}
}
