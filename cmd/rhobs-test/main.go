package main

import (
	"log"
	"strings"

	"flag"
	"time"

	"github.com/vprashar2929/rhobs-test/pkg/client"
	"github.com/vprashar2929/rhobs-test/pkg/deployment"
	"github.com/vprashar2929/rhobs-test/pkg/service"
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
	errList    []error
)

func main() {
	flag.StringVar(&namespace, "namespaces", defaultNamespace, "Namespace to be monitored")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "path of kubeconfig file")
	flag.DurationVar(&interval, "interval", defaultInterval, "Wait before retry status check again")
	flag.DurationVar(&timeout, "timeout", defaultTimeout, "Timeout for retry")
	flag.Parse()
	nsList := strings.Split(namespace, ",")
	clientset := client.GetClient(kubeconfig)
	err := deployment.CheckDeployments(nsList, clientset, interval, timeout)
	if err != nil {
		log.Printf("cannot validate deployements. reason: %v\n", err)
		errList = append(errList, err)
	}
	err = statefulset.CheckStatefulSets(nsList, clientset, interval, timeout)
	if err != nil {
		log.Printf("cannot validate statefulsets. reason: %v\n", err)
		errList = append(errList, err)
	}
	err = service.CheckServices(nsList, clientset, interval, timeout)
	if err != nil {
		log.Printf("cannot validate services. reason: %v\n", err)
		errList = append(errList, err)
	}
	if len(errList) > 0 {
		log.Fatalf("error occured while running tests")
	}
}
