package main

import (
	"os"

	"github.com/coreos/go-iptables/iptables"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
)

var rootCmd = &cobra.Command{
	Use:    "msm-iptables",
	Short:  "Set up iptables rules for an MSM Sidecar",
	Long:   "msm-iptables is responsible for setting up port forwarding for an MSM Sidecar.",
	PreRun: bindFlags,
	Run: func(cmd *cobra.Command, args []string) {

		ipt, err := iptables.New()
		if err != nil {
			handleErrorWithCode(err, 1)
		}

		//iptables -t nat -A OUTPUT -d 127.0.0.0/8 -j RETURN
		noRedirDestAddrRuleSpec := []string{"-d", viper.GetString(noRedirectDestAddr), "-j", "RETURN"}
		err = ipt.Append("nat", "OUTPUT", noRedirDestAddrRuleSpec...)
		if err != nil {
			handleErrorWithCode(err, 1)
		}

		//iptables -t nat -A OUTPUT -p tcp -m owner --uid-owner 1337 -j RETURN
		uidRulespec := []string{"-p", "tcp", "-m", "owner", "--uid-owner", viper.GetString(proxyUID), "-j", "RETURN"}
		err = ipt.Append("nat", "OUTPUT", uidRulespec...)
		if err != nil {
			handleErrorWithCode(err, 1)
		}

		//iptables -t nat -A OUTPUT -p tcp --dport 554 -j REDIRECT --to-ports 8554
		msmProxyPortRulespec := []string{"-p", "tcp", "--dport", defaultRTSPPort, "-j", redirectModeREDIRECT,
			"--to-ports", viper.GetString(msmProxyPort)}
		err = ipt.Append("nat", "OUTPUT", msmProxyPortRulespec...)
		if err != nil {
			handleErrorWithCode(err, 1)
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		handleError(err)
	}
}

func handleError(err error) {
	handleErrorWithCode(err, 1)
}

func handleErrorWithCode(err error, code int) {
	log.Error(err)
	os.Exit(code)
}

// Any viper mutation and binding should be placed in `PreRun` since they should be dynamically bound to the subcommand being executed.
func bindFlags(cmd *cobra.Command, args []string) {

	if err := viper.BindPFlag(msmProxyPort, cmd.Flags().Lookup(msmProxyPort)); err != nil {
		handleError(err)
	}
	viper.SetDefault(msmProxyPort, defaultRedirectToPort)

	if err := viper.BindPFlag(proxyUID, cmd.Flags().Lookup(proxyUID)); err != nil {
		handleError(err)
	}
	viper.SetDefault(proxyUID, "")

	if err := viper.BindPFlag(noRedirectDestAddr, cmd.Flags().Lookup(noRedirectDestAddr)); err != nil {
		handleError(err)
	}
	viper.SetDefault(noRedirectDestAddr, "")

	if err := viper.BindPFlag(inboundInterceptMode, cmd.Flags().Lookup(inboundInterceptMode)); err != nil {
		handleError(err)
	}
	viper.SetDefault(inboundInterceptMode, "")
}

func init() {
	rootCmd.Flags().StringP(msmProxyPort, "p", "", "Specify the msm port to which redirect all RTSP traffic (default: 8554)")

	rootCmd.Flags().StringP(proxyUID, "u", "", "UID of the user for which the redirection is not applied. The UID of the proxy container")

	rootCmd.Flags().StringP(noRedirectDestAddr, "d", "", "The localhost address to return outbound traffic")

	rootCmd.Flags().StringP(inboundInterceptMode, "m", "",
		"The mode used to redirect inbound connections to MSM Proxy, default: REDIRECT")
}
