package install

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/pkg/errors"

	"github.com/media-streaming-mesh/msm-cni/util"
	log "github.com/sirupsen/logrus"
)

var rootCmd = &cobra.Command{
	Use:   "cni-installer",
	Short: "Install and configure MSM CNI plugin on a node",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()

		var cfg *Config
		if cfg, err = constructConfig(); err != nil {
			return
		}
		log.Infof("install msm-cni, configuration: \n%+v", cfg)

		isReady := StartServer()

		installer := NewInstaller(cfg, isReady)

		if err = installer.Run(ctx); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				err = nil
			}
		}

		if cleanErr := installer.Cleanup(); cleanErr != nil {
			if err != nil {
				err = errors.Wrap(err, cleanErr.Error())
			} else {
				err = cleanErr
			}
		}

		return
	},
}

// GetCommand returns the main cobra.Command object for this application
func GetCommand() *cobra.Command {
	return rootCmd
}

func init() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	registerStringParameter(CNINetDir, "/etc/cni/net.d", "Directory on the host where CNI networks are installed")
	registerStringParameter(CNIConfName, "", "Name of the CNI configuration file")
	registerBooleanParameter(ChainedCNIPlugin, true, "Whether to install CNI plugin as a chained or standalone")
	registerStringParameter(CNINetworkConfig, "", "CNI config template as a string")
	registerStringParameter(LogLevel, "warn", "Fallback value for log level in CNI config file, if not specified in helm template")

	// Not configurable in CNI helm charts
	registerStringParameter(MountedCNINetDir, "/host/etc/cni/net.d", "Directory on the container where CNI networks are installed")
	registerStringParameter(CNINetworkConfigFile, "", "CNI config template as a file")
	registerStringParameter(KubeconfigFilename, "ZZZ-msm-cni-kubeconfig", "Name of the kubeconfig file")
	registerIntegerParameter(KubeconfigMode, DefaultKubeconfigMode, "File mode of the kubeconfig file")
	registerStringParameter(KubeCAFile, "", "CA file for kube Defaults to the pod one")
	registerBooleanParameter(SkipTLSVerify, false, "Whether to use insecure TLS in kubeconfig file")
	registerBooleanParameter(UpdateCNIBinaries, true, "Update binaries")
	registerStringArrayParameter(SkipCNIBinaries, []string{}, "Binaries that should not be installed")
}

func registerStringParameter(name, value, usage string) {
	rootCmd.Flags().String(name, value, usage)
	bindViper(name)
}

func registerStringArrayParameter(name string, value []string, usage string) {
	rootCmd.Flags().StringArray(name, value, usage)
	bindViper(name)
}

func registerIntegerParameter(name string, value int, usage string) {
	rootCmd.Flags().Int(name, value, usage)
	bindViper(name)
}

func registerBooleanParameter(name string, value bool, usage string) {
	rootCmd.Flags().Bool(name, value, usage)
	bindViper(name)
}

func bindViper(name string) {
	if err := viper.BindPFlag(name, rootCmd.Flags().Lookup(name)); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

func constructConfig() (*Config, error) {
	cfg := &Config{
		CNINetDir:        viper.GetString(CNINetDir),
		MountedCNINetDir: viper.GetString(MountedCNINetDir),
		CNIConfName:      viper.GetString(CNIConfName),
		ChainedCNIPlugin: viper.GetBool(ChainedCNIPlugin),

		CNINetworkConfigFile: viper.GetString(CNINetworkConfigFile),
		CNINetworkConfig:     viper.GetString(CNINetworkConfig),

		LogLevel:           viper.GetString(LogLevel),
		KubeconfigFilename: viper.GetString(KubeconfigFilename),
		KubeconfigMode:     viper.GetInt(KubeconfigMode),
		KubeCAFile:         viper.GetString(KubeCAFile),
		SkipTLSVerify:      viper.GetBool(SkipTLSVerify),
		K8sServiceProtocol: os.Getenv("KUBERNETES_SERVICE_PROTOCOL"),
		K8sServiceHost:     os.Getenv("KUBERNETES_SERVICE_HOST"),
		K8sServicePort:     os.Getenv("KUBERNETES_SERVICE_PORT"),
		K8sNodeName:        os.Getenv("KUBERNETES_NODE_NAME"),

		CNIBinSourceDir:   CNIBinDir,
		CNIBinTargetDirs:  []string{HostCNIBinDir, SecondaryBinDir},
		UpdateCNIBinaries: viper.GetBool(UpdateCNIBinaries),
		SkipCNIBinaries:   viper.GetStringSlice(SkipCNIBinaries),
	}

	if len(cfg.K8sNodeName) == 0 {
		var err error
		cfg.K8sNodeName, err = os.Hostname()
		if err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

type Installer struct {
	cfg                *Config
	isReady            *atomic.Value
	saToken            string
	kubeconfigFilepath string
	cniConfigFilepath  string
}

// NewInstaller returns an instance of Installer with the given config
func NewInstaller(cfg *Config, isReady *atomic.Value) *Installer {
	return &Installer{
		cfg:     cfg,
		isReady: isReady,
	}
}

// Run starts the installation process, verifies the configuration, then sleeps.
// If an invalid configuration is detected, the installation process will restart to restore a valid state.
func (in *Installer) Run(ctx context.Context) (err error) {
	for {
		if err = copyBinaries(
			in.cfg.CNIBinSourceDir, in.cfg.CNIBinTargetDirs,
			in.cfg.UpdateCNIBinaries, in.cfg.SkipCNIBinaries); err != nil {
			return
		}

		if in.saToken, err = readServiceAccountToken(); err != nil {
			return
		}

		if in.kubeconfigFilepath, err = createKubeconfigFile(in.cfg, in.saToken); err != nil {
			return
		}

		if in.cniConfigFilepath, err = createCNIConfigFile(ctx, in.cfg, in.saToken); err != nil {
			return
		}

		if err = sleepCheckInstall(ctx, in.cfg, in.cniConfigFilepath, in.isReady); err != nil {
			return
		}
		// Invalid config; pod set to "NotReady"
		log.Info("Restarting...")
	}
}

// Cleanup remove MSM CNI's config, kubeconfig file, and binaries.
func (in *Installer) Cleanup() error {
	log.Info("Cleaning up.")
	if len(in.cniConfigFilepath) > 0 && util.Exists(in.cniConfigFilepath) {
		if in.cfg.ChainedCNIPlugin {
			log.Infof("Removing MSM CNI config from CNI config file: %s", in.cniConfigFilepath)

			// Read JSON from CNI config file
			cniConfigMap, err := util.ReadCNIConfigMap(in.cniConfigFilepath)
			if err != nil {
				return err
			}
			// Find MSM CNI and remove from plugin list
			plugins, err := util.GetPlugins(cniConfigMap)
			if err != nil {
				return errors.Wrap(err, in.cniConfigFilepath)
			}
			for i, rawPlugin := range plugins {
				plugin, err := util.GetPlugin(rawPlugin)
				if err != nil {
					return errors.Wrap(err, in.cniConfigFilepath)
				}
				if plugin["type"] == "msm-cni" {
					cniConfigMap["plugins"] = append(plugins[:i], plugins[i+1:]...)
					break
				}
			}

			cniConfig, err := util.MarshalCNIConfig(cniConfigMap)
			if err != nil {
				return err
			}
			if err = util.AtomicWrite(in.cniConfigFilepath, cniConfig, os.FileMode(0644)); err != nil {
				return err
			}
		} else {
			log.Infof("Removing MSM CNI config file: %s", in.cniConfigFilepath)
			if err := os.Remove(in.cniConfigFilepath); err != nil {
				return err
			}
		}
	}

	if len(in.kubeconfigFilepath) > 0 && util.Exists(in.kubeconfigFilepath) {
		log.Infof("Removing MSM CNI kubeconfig file: %s", in.kubeconfigFilepath)
		if err := os.Remove(in.kubeconfigFilepath); err != nil {
			return err
		}
	}

	for _, targetDir := range in.cfg.CNIBinTargetDirs {
		if msmCNIBin := filepath.Join(targetDir, "msm-cni"); util.Exists(msmCNIBin) {
			log.Infof("Removing binary: %s", msmCNIBin)
			if err := os.Remove(msmCNIBin); err != nil {
				return err
			}
		}
		if msmIptablesBin := filepath.Join(targetDir, "msm-iptables"); util.Exists(msmIptablesBin) {
			log.Infof("Removing binary: %s", msmIptablesBin)
			if err := os.Remove(msmIptablesBin); err != nil {
				return err
			}
		}
	}
	return nil
}

func readServiceAccountToken() (string, error) {
	saToken := ServiceAccountPath + "/token"
	if !util.Exists(saToken) {
		return "", fmt.Errorf("service account token file %s does not exist. Is this not running within a pod?", saToken)
	}

	token, err := ioutil.ReadFile(saToken)
	if err != nil {
		return "", err
	}

	return string(token), nil
}

// sleepCheckInstall verifies the configuration then blocks until an invalid configuration is detected, and return nil.
// If an error occurs or context is canceled, the function will return the error.
// Returning from this function will set the pod to "NotReady".
func sleepCheckInstall(ctx context.Context, cfg *Config, cniConfigFilepath string, isReady *atomic.Value) error {
	// Create file watcher before checking for installation
	// so that no file modifications are missed while and after checking
	watcher, fileModified, errChan, err := util.CreateFileWatcher(cfg.MountedCNINetDir)
	if err != nil {
		return err
	}
	defer func() {
		SetNotReady(isReady)
		_ = watcher.Close()
	}()

	for {
		if checkErr := checkInstall(cfg, cniConfigFilepath); checkErr != nil {
			// Pod set to "NotReady" due to invalid configuration
			log.Infof("Invalid configuration. %v", checkErr)
			return nil
		}
		// Check if file has been modified or if an error has occurred during checkInstall before setting isReady to true
		select {
		case <-fileModified:
			return nil
		case err := <-errChan:
			return err
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Valid configuration; set isReady to true and wait for modifications before checking again
			SetReady(isReady)
			if err = util.WaitForFileMod(ctx, fileModified, errChan); err != nil {
				// Pod set to "NotReady" before termination
				return err
			}
		}
	}
}

// checkInstall returns an error if an invalid CNI configuration is detected
func checkInstall(cfg *Config, cniConfigFilepath string) error {
	defaultCNIConfigFilename, err := getDefaultCNINetwork(cfg.MountedCNINetDir)
	if err != nil {
		return err
	}
	defaultCNIConfigFilepath := filepath.Join(cfg.MountedCNINetDir, defaultCNIConfigFilename)
	if defaultCNIConfigFilepath != cniConfigFilepath {
		if len(cfg.CNIConfName) > 0 {
			// Install was run with overridden CNI config file so don't error out on preempt check
			// Likely the only use for this is testing the script
			log.Warnf("CNI config file %s preempted by %s", cniConfigFilepath, defaultCNIConfigFilepath)
		} else {
			return fmt.Errorf("CNI config file %s preempted by %s", cniConfigFilepath, defaultCNIConfigFilepath)
		}
	}

	if !util.Exists(cniConfigFilepath) {
		return fmt.Errorf("CNI config file removed: %s", cniConfigFilepath)
	}

	if cfg.ChainedCNIPlugin {
		// Verify that MSM CNI config exists in the CNI config plugin list
		cniConfigMap, err := util.ReadCNIConfigMap(cniConfigFilepath)
		if err != nil {
			return err
		}
		plugins, err := util.GetPlugins(cniConfigMap)
		if err != nil {
			return errors.Wrap(err, cniConfigFilepath)
		}
		for _, rawPlugin := range plugins {
			plugin, err := util.GetPlugin(rawPlugin)
			if err != nil {
				return errors.Wrap(err, cniConfigFilepath)
			}
			if plugin["type"] == "msm-cni" {
				return nil
			}
		}

		return fmt.Errorf("msm-cni CNI config removed from CNI config file: %s", cniConfigFilepath)
	}
	// Verify that MSM CNI config exists as a standalone plugin
	cniConfigMap, err := util.ReadCNIConfigMap(cniConfigFilepath)
	if err != nil {
		return err
	}

	if cniConfigMap["type"] != "msm-cni" {
		return fmt.Errorf("msm-cni CNI config file modified: %s", cniConfigFilepath)
	}
	return nil
}
