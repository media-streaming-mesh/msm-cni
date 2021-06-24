package main

// Constants used as default values cobra/viper CLI
const (
	redirectModeREDIRECT      = "REDIRECT"
	defaultRedirectMode       = redirectModeREDIRECT
	defaultRedirectToPort     = "8554"
	defaultRTSPPort           = "554"
	defaultNoRedirectUID      = "1337"
	defaultNoRedirectDestAddr = "127.0.0.0/8"
)

// Constants used in cobra/viper CLI
const (
	msmProxyPort         = "msm-proxy-port"
	proxyUID             = "proxy-uid"
	noRedirectDestAddr   = "redir-dest-addr"
	inboundInterceptMode = "inbound-intercept-mode"
)
