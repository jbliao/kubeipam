package cni

import (
	"github.com/containernetworking/cni/pkg/types"
)

// IPAMConf extend official's IPAM config
type IPAMConf struct {
	types.IPAM
	KubeConfigPath string   `json:"configPath"`
	PoolName       string   `json:"poolName"`
	PoolNamespace  string   `json:"poolNamespace"`
	Mask           string   `json:"mask"`
	Gateway        string   `json:"gateway"`
	Routes         []string `json:"routes"`
	LogFile        string   `json:"logFile"`
}

// PluginConf extend official's cni conf, but use custom ipamconf
type PluginConf struct {
	types.NetConf
	IPAM IPAMConf `json:"ipam"`
}
