package main

import (
	"strings"

	"flag"
	"time"

	"github.com/vprashar2929/rhobs-test/pkg/client"
	"github.com/vprashar2929/rhobs-test/pkg/deployment"
	"github.com/vprashar2929/rhobs-test/pkg/statefulset"
)

const (
	defaultInterval  = 1 * time.Minute
	defaultTimeout   = 5 * time.Minute
	defaultNamespace = "default"
)

var (
	namespace  string
	kubeconfig string
	interval   time.Duration
	timeout    time.Duration
)

func main() {
	flag.StringVar(&namespace, "namespaces", defaultNamespace, "Namespace to be monitored")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "path of kubeconfig file")
	flag.DurationVar(&interval, "interval", defaultInterval, "Wait before retry status check again")
	flag.DurationVar(&timeout, "timeout", defaultTimeout, "Timeout for retry")
	flag.Parse()
	nsList := strings.Split(namespace, ",")
	clientset := client.GetClient(kubeconfig)
	deployment.CheckDeployments(nsList, clientset, interval, timeout)
	statefulset.CheckStatefulSets(nsList, clientset, interval, timeout)

}
