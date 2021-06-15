package install

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/media-streaming-mesh/msm-cni/util"

	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
)

const kubeconfigTemplate = `# Kubeconfig file for MSM CNI plugin.
apiVersion: v1
kind: Config
clusters:
- name: local
  cluster:
    server: {{.KubernetesServiceProtocol}}://[{{.KubernetesServiceHost}}]:{{.KubernetesServicePort}}
    {{.TLSConfig}}
users:
- name: msm-cni
  user:
    token: "{{.ServiceAccountToken}}"
contexts:
- name: msm-cni-context
  context:
    cluster: local
    user: msm-cni
current-context: msm-cni-context
`

type kubeconfigFields struct {
	KubernetesServiceProtocol string
	KubernetesServiceHost     string
	KubernetesServicePort     string
	ServiceAccountToken       string
	TLSConfig                 string
}

func createKubeconfigFile(cfg *Config, saToken string) (kubeconfigFilepath string, err error) {
	if len(cfg.K8sServiceHost) == 0 {
		err = errors.New("KUBERNETES_SERVICE_HOST not set. Is this not running within a pod?")
		return
	}

	if len(cfg.K8sServicePort) == 0 {
		err = errors.New("KUBERNETES_SERVICE_PORT not set. Is this not running within a pod?")
		return
	}

	var tpl *template.Template
	tpl, err = template.New("kubeconfig").Parse(kubeconfigTemplate)
	if err != nil {
		return
	}

	protocol := cfg.K8sServiceProtocol
	if protocol == "" {
		protocol = "https"
	}

	caFile := cfg.KubeCAFile
	if caFile == "" {
		caFile = ServiceAccountPath + "/ca.crt"
	}

	var tlsConfig string
	if cfg.SkipTLSVerify {
		tlsConfig = "insecure-skip-tls-verify: true"
	} else {
		if !util.Exists(caFile) {
			return "", fmt.Errorf("file does not exist: %s", caFile)
		}
		var caContents []byte
		caContents, err = ioutil.ReadFile(caFile)
		if err != nil {
			return
		}
		caBase64 := base64.StdEncoding.EncodeToString(caContents)
		tlsConfig = "certificate-authority-data: " + caBase64
	}

	fields := kubeconfigFields{
		KubernetesServiceProtocol: protocol,
		KubernetesServiceHost:     cfg.K8sServiceHost,
		KubernetesServicePort:     cfg.K8sServicePort,
		ServiceAccountToken:       saToken,
		TLSConfig:                 tlsConfig,
	}

	var kcbb bytes.Buffer
	if err := tpl.Execute(&kcbb, fields); err != nil {
		return "", err
	}

	var kcbbToPrint bytes.Buffer
	fields.ServiceAccountToken = "<redacted>"
	if !cfg.SkipTLSVerify {
		fields.TLSConfig = fmt.Sprintf("certificate-authority-data: <CA cert from %s>", caFile)
	}
	if err := tpl.Execute(&kcbbToPrint, fields); err != nil {
		return "", err
	}

	kubeconfigFilepath = filepath.Join(cfg.MountedCNINetDir, cfg.KubeconfigFilename)
	log.Infof("write kubeconfig file %s with: \n%+v", kubeconfigFilepath, kcbbToPrint.String())
	if err = util.AtomicWrite(kubeconfigFilepath, kcbb.Bytes(), os.FileMode(cfg.KubeconfigMode)); err != nil {
		return "", err
	}

	return
}
