# MSM CNI plugin

Any application pod that is MSM enabled will have all its traffic to/from the pods go through the
MSM stub (sidecar proxy).  The MSM CNI works as a chained plugin to the already installed CNIs 
(that provide network connectivity to the pods) and is responsible is to install all the rules without 
the need to give privileged access to the application pods.

The current implementation is configuring the iptables rules in the netns for the pods. MSM CNI runs as a DaemonSet 
on a Kubernetes cluster (runs on every node) and can be configured via a configuration file. 

## Usage

The easiest way to get started with the MSM CNI is by using the deployment example
found under [MSM CNI Helm chart](https://github.com/media-streaming-mesh/deployments-kubernetes/blob/master/examples/features/cni/cni.yaml)

## Implementation Details

### Overview

- [MSM CNI Helm chart](https://github.com/media-streaming-mesh/deployments-kubernetes/blob/master/examples/features/cni/cni.yaml)
    - `msm-cni` daemonset
    - `msm-cni-config` chained CNI configuration for MSM CNI
    - creates service-account `msm-cni` and `ClusterRoleBinding` to allow GET queries for pods from K8s API

- `installer` container
    - creates kubeconfig for the service account the pod runs under
    - copies the binaries `msm-cni`and `msm-iptables` `/opt/cni/bin`
    - appends the MSM CNI plugin configuration to any already installed CNI configuration file

- `msm-cni`
    - a CNI plugin executable
    - on pod add, decides if pod should redirect traffic to MSM stub (sidecar proxy) by installing iptables rules

- `msm-iptables`
    - an executable responsible to set up iptables to redirect a list of ports to the MSM sidecar proxy
    
## Troubleshooting

### Collecting Logs

The CNI plugins are executed by threads in the `kubelet` process.  The CNI plugins logs can be found
under the `kubelet` process. An example to view the last 1000 lines of the kubelet process is:

```console
$ journalctl -t kubelet -n 1000 | less
```