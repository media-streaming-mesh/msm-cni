package install

const (
	MountedCNINetDir     = "mounted-cni-net-dir"
	CNINetDir            = "cni-net-dir"
	CNIConfName          = "cni-conf-name"
	ChainedCNIPlugin     = "chained-cni-plugin"
	CNINetworkConfigFile = "cni-network-config-file"
	CNINetworkConfig     = "cni-network-config"
	LogLevel             = "log-level"
	KubeconfigFilename   = "kubecfg-file-name"
	KubeconfigMode       = "kubeconfig-mode"
	KubeCAFile           = "kube-ca-file"
	SkipTLSVerify        = "skip-tls-verify"
	SkipCNIBinaries      = "skip-cni-binaries"
	UpdateCNIBinaries    = "update-cni-binaries"
)

// Internal constants
var (
	CNIBinDir             = "/opt/cni/bin"
	HostCNIBinDir         = "/host/opt/cni/bin"
	SecondaryBinDir       = "/host/secondary-bin-dir"
	ServiceAccountPath    = "/var/run/secrets/kubernetes.io/serviceaccount"
	DefaultKubeconfigMode = 0600

	// K8s liveness and readiness endpoints
	LivenessEndpoint  = "/healthz"
	ReadinessEndpoint = "/readyz"
	Port              = "8000"
)
