// Defines the redirect object and operations.
package cni

const (
	redirectModeREDIRECT      = "REDIRECT"
	defaultRedirectToPort     = "8554"
	defaultRedirectMode       = redirectModeREDIRECT
	defaultNoRedirectUID      = "1337"
	defaultNoRedirectDestAddr = "127.0.0.0/8"
)

// Redirect is the msm-cni redirect object
type Redirect struct {
	targetPort         string
	redirectMode       string
	noRedirectUID      string
	noRedirectDestAddr string
}

// NewRedirect returns a new Redirect Object constructed from a list of ports and annotations
// For now we are using some default values but that can change in the future to support
// passing values through annotations
func NewRedirect(_ *PodInfo) (*Redirect, error) {
	return &Redirect{
		targetPort:         defaultRedirectToPort,
		redirectMode:       defaultRedirectMode,
		noRedirectUID:      defaultNoRedirectUID,
		noRedirectDestAddr: defaultNoRedirectDestAddr,
	}, nil
}
