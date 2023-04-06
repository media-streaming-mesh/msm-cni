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
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/pkg/version"
	log "github.com/sirupsen/logrus"
)

// parseConfig parses the supplied configuration (and prevResult) from stdin.
func parseConfig(stdin []byte) (*PluginConf, error) {
	conf := PluginConf{}

	if err := json.Unmarshal(stdin, &conf); err != nil {
		return nil, fmt.Errorf("failed to parse network configuration: %v", err)
	}

	// Parse previous CNI config result. This is for when the CNI plugin is chained
	if conf.RawPrevResult != nil {
		resultBytes, err := json.Marshal(conf.RawPrevResult)
		if err != nil {
			return nil, fmt.Errorf("could not serialize prevResult: %v", err)
		}
		res, err := version.NewResult(conf.CNIVersion, resultBytes)
		if err != nil {
			return nil, fmt.Errorf("could not parse prevResult: %v", err)
		}
		conf.RawPrevResult = nil
		conf.PrevResult, err = current.NewResultFromResult(res)
		if err != nil {
			return nil, fmt.Errorf("could not convert result to current version: %v", err)
		}
	}
	// End previous result parsing

	return &conf, nil
}

// CmdAdd is called for pod ADD requests
func CmdAdd(args *skel.CmdArgs) error {
	// open a file
	f, err := os.OpenFile("/var/log/testlogrus.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		fmt.Printf("error opening file: %v", err)
	}
	// don't forget to close it
	defer f.Close()

	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})

	// Output to stderr instead of stdout, could also be a file.
	log.SetOutput(f)

	log.Infof("got into cmdadd")
	// Defer a panic recover, so that in case if panic we can still return
	// a proper error to the runtime.
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("msm-cni cmdAdd error: %v", err)
		}
	}()

	log.Infof("before parse")

	conf, err := parseConfig(args.StdinData)
	if err != nil {
		log.Errorf("error parsing msm-cni cmdAdd config: %v", err)
		return err
	}

	log.Infof("after parse")

	var loggedPrevResult interface{}
	if conf.PrevResult == nil {
		loggedPrevResult = "none"
	} else {
		loggedPrevResult = conf.PrevResult
	}

	log.Infof("CmdAdd config parsed with values, version=%s, prevResult=%v", conf.CNIVersion, loggedPrevResult)

	// Determine if running under k8s by checking the CNI args
	k8sArgs := KubernetesArgs{}
	if err := types.LoadArgs(args.Args, &k8sArgs); err != nil {
		return err
	}
	log.Infof("Getting identifiers with arguments: %s", args.Args)
	log.Infof("Loaded k8s arguments: %v", k8sArgs)

	if conf.Kubernetes.CNIBinDir != "" {
		nsSetupBinDir = conf.Kubernetes.CNIBinDir
	}
	if conf.Kubernetes.InterceptRuleMgrType != "" {
		interceptRuleMgrType = conf.Kubernetes.InterceptRuleMgrType
	}

	// Check if the workload is running under Kubernetes.
	if string(k8sArgs.K8S_POD_NAMESPACE) != "" && string(k8sArgs.K8S_POD_NAME) != "" {
		excludePod := false
		// check if pod belongs to an excluded namespace defined in the plugin configuration
		for _, excludeNs := range conf.Kubernetes.ExcludeNamespaces {
			if string(k8sArgs.K8S_POD_NAMESPACE) == excludeNs {
				excludePod = true
				break
			}
		}

		if !excludePod {
			// create a kubernetes API client
			client, err := newKubeClient(*conf)
			if err != nil {
				log.Errorf("Failed to create kubernetes client, err=%v", err)
				return err
			}

			podInfo := &PodInfo{}
			for retry := 1; retry <= podRetrievalMaxRetries; retry++ {
				podInfo, err = getKubePodInfo(client, string(k8sArgs.K8S_POD_NAME), string(k8sArgs.K8S_POD_NAMESPACE))
				if err == nil {
					break
				}
				log.Warnf("Waiting for pod metadata, err=%v, retry=%d/%d", err, retry, podRetrievalMaxRetries)
				time.Sleep(podRetrievalInterval)
			}
			// error after reaching max retries number
			if err != nil {
				log.Errorf("Failed to get pod data, err=%v", err)
				return err
			}

			if len(podInfo.Containers) >= 1 {
				log.Infof("Found containers %v", podInfo.Containers)

				// check annotations before invoking redirect commands
				if _, ok := podInfo.Annotations[sidecarAnnotationKey]; !ok {
					log.Infof("Pod %s excluded - no sidecar annotation", string(k8sArgs.K8S_POD_NAME))
					excludePod = true
				}

				if !excludePod {
					log.Infof("setting up redirect")

					redirect, err := NewRedirect(podInfo)
					if err != nil {
						log.Errorf("Pod redirect failed due to bad params: %v", err)
						return err
					} else {
						intMgrCt := GetInterceptRuleMgrCtor(interceptRuleMgrType)
						if intMgrCt == nil {
							log.Errorf("Pod redirect failed due to unavailable InterceptRuleMgr of type %s",
								interceptRuleMgrType)
						} else {
							rulesMgr := intMgrCt()
							if err := rulesMgr.Program(args.Netns, redirect); err != nil {
								return err
							}
						}
					}
				}
			}
		} else {
			log.Infof("Pod is excluded from msm-cni")
		}
	} else {
		log.Infof("Pod is not running under Kubernetes")
	}

	var result *current.Result
	if conf.PrevResult == nil {
		result = &current.Result{
			CNIVersion: current.ImplementedSpecVersion,
		}
	} else {
		// Pass through the result for the next plugin
		result = conf.PrevResult
	}

	return types.PrintResult(result, conf.CNIVersion)
}

// CmdGet is called for pod Get requests
func CmdGet(args *skel.CmdArgs) error {
	log.Info("CmdGet not implemented")
	return fmt.Errorf("CmdGet not implemented")
}

// cmdDel is called for pod DELETE requests
func CmdDel(args *skel.CmdArgs) error {
	// nothing to cleanup for msm-cni, everything is happening on pod level
	return nil
}
