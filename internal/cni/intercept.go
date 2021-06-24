package cni

const (
	defInterceptRuleMgrType = "iptables"
)

// InterceptRuleMgr configures networking tables (e.g. iptables or nftables) for
// redirecting traffic to an MSM proxy.
type InterceptRuleMgr interface {
	Program(netns string, redirect *Redirect) error
}

type InterceptRuleMgrCtor func() InterceptRuleMgr

var InterceptRuleMgrTypes = map[string]InterceptRuleMgrCtor{
	"iptables": IptablesInterceptRuleMgrCtor,
}

// Constructor factory for known types of InterceptRuleMgr's
func GetInterceptRuleMgrCtor(interceptType string) InterceptRuleMgrCtor {
	return InterceptRuleMgrTypes[interceptType]
}

// Constructor for iptables InterceptRuleMgr
func IptablesInterceptRuleMgrCtor() InterceptRuleMgr {
	return newIPTables()
}
