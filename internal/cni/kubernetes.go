/*
 * Copyright (c) 2022 Cisco and/or its affiliates.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
	interceptRuleMgrType   = defInterceptRuleMgrType
	podRetrievalMaxRetries = 30
	podRetrievalInterval   = 1 * time.Second
)

const msmSideCarLabel = "sidecar.mediastreamingmesh.io/inject"

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
	RawPrevResult *map[string]interface{} `json:"prevResult"`
	PrevResult    *current.Result         `json:"-"`

	// Plugin-specific flags
	LogLevel   string     `json:"logLevel"`
	LogType    string     `json:"logType"`
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

	for _, initContainer := range pod.Spec.InitContainers {
		podInfo.InitContainers[initContainer.Name] = struct{}{}
	}
	for containerIdx, container := range pod.Spec.Containers {
		log.Debugf("Inspecting container, pod=%s, container=%s", pod, podName)
		podInfo.Containers[containerIdx] = container.Name
	}

	return podInfo, nil
}
