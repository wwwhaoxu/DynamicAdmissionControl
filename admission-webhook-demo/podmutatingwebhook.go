package main

import (
	"context"
	"encoding/json"
	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type podMutate struct {
	Client client.Client
	decode *admission.Decoder
}

type configMap struct {
	Containers []corev1.Container
	Volumes    []corev1.Volume
}

func (p *podMutate) Handle(ctx context.Context, req admission.Request) admission.Response {

	podMutateLog := log.WithName("podMutate")
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Load the Envoy Initializer configuration from a Kubernetes ConfigMap.
	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configmap, metav1.GetOptions{})
	if err != nil {
		klog.Fatal()
	}

	c, err := configmapToConfig(cm)
	if err != nil {
		klog.Fatal(err)
	}

	pod := &corev1.Pod{}
	//initializePod := runtime.DeepCopyJSONValue(pod)

	err = p.decode.Decode(req, pod)
	if err != nil {
		podMutateLog.Error(err, "failed decoder pod")
		return admission.Errored(http.StatusBadRequest, err)
	}
	//initializedPod := initializePod.(*corev1.Pod)

	pod.Spec.Containers = append(pod.Spec.Containers, c.Containers...)
	pod.Spec.Volumes = append(pod.Spec.Volumes, c.Volumes...)

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		podMutateLog.Error(err, "failed marshal pod")
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// InjectDecoder injects the decoder
func (p *podMutate) InjectDecoder(d *admission.Decoder) error {
	p.decode = d
	return nil
}

func configmapToConfig(configmap *corev1.ConfigMap) (*configMap, error) {
	var c configMap
	err := yaml.Unmarshal([]byte(configmap.Data["config"]), &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
