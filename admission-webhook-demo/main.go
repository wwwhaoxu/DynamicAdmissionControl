package main

import (
	"flag"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	defaultConfigmap = "envoy-initializer"
	defaultNamespace = "default"
)

var (
	configmap   string
	namespace   string
	metricsAddr string
	port        int
	scheme      = runtime.NewScheme()
	log         = logf.Log.WithName("pod-admission-webhook")
)

func init() {
	logf.SetLogger(zap.New())
}

func main() {

	entryLog := log.WithName("entrypoint")

	flag.StringVar(&configmap, "configmap", defaultConfigmap, "The envoy initializer configuration configmap")
	flag.StringVar(&namespace, "namespace", defaultNamespace, "The configuration namespace")

	pflag.IntVar(&port, "port", 9443, "pod-admission-webhook listen port.")
	pflag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	//Setup a Manager
	entryLog.Info("setting up manager")
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               port,
	})

	if err != nil {
		entryLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	//Setup webhooks
	entryLog.Info("setting up webhook server")
	hookServer := mgr.GetWebhookServer()

	entryLog.Info("Registering webhook to the webhook server")
	hookServer.Register("/mutate-pod", &webhook.Admission{Handler: &podMutate{Client: mgr.GetClient()}})

	entryLog.Info("staring manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		entryLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
