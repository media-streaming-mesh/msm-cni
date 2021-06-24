package cni

import (
	"context"
	"net"
	"time"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	log "github.com/sirupsen/logrus"
)

var (
	nsSetupBinDir          = "/opt/cni/bin"
	sidecarAnnotationKey   = "sidecar.mediastreamingmesh.io/inject"
	interceptRuleMgrType   = defInterceptRuleMgrType
	podRetrievalMaxRetries = 30
	podRetrievalInterval   = 1 * time.Second
)

// Kubernetes a K8s specific struct to hold config
type Kubernetes struct {
	K8sAPIRoot           string   `json:"kubernetesAPIRoot"`
	KubeConfig           string   `json:"kubeConfig"`
	InterceptRuleMgrType string   `json:"interceptName"`
	NodeName             string   `json:"nodeName"`
	ExcludeNamespaces    []string `json:"excludeNamespaces"`
	CNIBinDir            string   `json:"cniBinDir"`
}

// PluginConf is the expected json configuration passed in on stdin.
type PluginConf struct {
	types.NetConf           // You may wish to not nest this type
	RuntimeConfig *struct{} `json:"runtimeConfig"`

	// Previous result, when called in the context of a chained plugin.
	RawPrevResult *map[string]interface{} `json:"rawPrevResult"`
	PrevResult    *current.Result         `json:"prevResult"`

	// Plugin-specific flags
	LogLevel   string     `json:"logLevel"`
	Kubernetes Kubernetes `json:"kubernetes"`
}

// KubernetesArgs is the valid CNI_ARGS used for Kubernetes
// The field names need to match exact keys in kubelet args for unmarshalling
type KubernetesArgs struct {
	types.CommonArgs
	IP                         net.IP
	K8S_POD_NAME               types.UnmarshallableString
	K8S_POD_NAMESPACE          types.UnmarshallableString
	K8S_POD_INFRA_CONTAINER_ID types.UnmarshallableString
}

// PodInfo holds the information of a Kubernetes pod
type PodInfo struct {
	Containers        []string
	InitContainers    map[string]struct{}
	Labels            map[string]string
	Annotations       map[string]string
	ProxyEnvironments map[string]string
}

// newKubeClient returns a Kubernetes client
func newKubeClient(conf PluginConf) (*kubernetes.Clientset, error) {
	// Some config can be passed in a kubeConfig file
	kubeConfig := conf.Kubernetes.KubeConfig

	// Config can be overridden by config passed in explicitly in the network config.
	configOverrides := &clientcmd.ConfigOverrides{}

	// Use the kubernetes client code to load the kubeConfig file and combine it with the overrides.
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{
			ExplicitPath: kubeConfig,
		},
		configOverrides,
	).ClientConfig()
	if err != nil {
		log.Infof("Failed setting up kubernetes client with kubeConfig %s", kubeConfig)
		return nil, err
	}

	log.Infof("Set up kubernetes client with kubeConfig %s", kubeConfig)
	log.Infof("Kubernetes config: %v", config)

	// Create and return the clientSet
	return kubernetes.NewForConfig(config)
}

// getKubePodInfo returns information of a POD
func getKubePodInfo(client *kubernetes.Clientset, podName, podNamespace string) (*PodInfo, error) {
	pod, err := client.CoreV1().Pods(podNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
	log.Infof("pod info: %+v", pod)
	if err != nil {
		log.Infof("could not get pod info: %+v", pod)
		return nil, err
	}

	podInfo := &PodInfo{
		InitContainers:    make(map[string]struct{}),
		Containers:        make([]string, len(pod.Spec.Containers)),
		Labels:            pod.Labels,
		Annotations:       pod.Annotations,
		ProxyEnvironments: make(map[string]string),
	}

	// when annotations we could do smart things here

	return podInfo, nil
}
