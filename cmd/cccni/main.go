package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/pkg/version"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
	"github.com/jbliao/kubeipam/pkg/cni"
	"github.com/jbliao/kubeipam/pkg/cni/allocator"
	"github.com/jbliao/kubeipam/pkg/cni/pool"
)

func main() {
	skel.PluginMain(cmdAdd, nil, cmdDel, version.All, bv.BuildString("cccni"))
}

func loadNetConf(bytes []byte) (*cni.PluginConf, error) {
	conf := &cni.PluginConf{}
	if err := json.Unmarshal(bytes, conf); err != nil {
		return nil, fmt.Errorf("failed to load netconf: %v", err)
	}

	if conf.IPAM.KubeConfigPath == "" ||
		conf.IPAM.PoolName == "" ||
		conf.IPAM.PoolNamespace == "" {
		return nil, fmt.Errorf("K8s API Config not given, Please check the cni ipam config")
	}
	return conf, nil
}

func cmdAdd(args *skel.CmdArgs) error {
	conf, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}
	log.Printf("cmdAdd begin")

	pool, err := pool.NewKubeIPAMPool(&conf.IPAM)
	if err != nil {
		return err
	}

	alctr, err := allocator.NewRoundRobinAllocator()
	if err != nil {
		return err
	}

	ip, err := alctr.Allocate(pool, args.ContainerID)
	if err != nil {
		return err
	}

	result := &current.Result{
		IPs: []*current.IPConfig{{
			Version: "4",
			Address: net.IPNet{
				IP:   ip,
				Mask: interface{}(net.ParseIP(conf.IPAM.Mask)).(net.IPMask),
			},
			Gateway: net.ParseIP(conf.IPAM.Gateway),
		}},
		Routes: []*types.Route{{
			Dst: net.IPNet{
				IP:   net.IPv4(0, 0, 0, 0),
				Mask: net.CIDRMask(0, 32),
			},
			GW: net.ParseIP(conf.IPAM.Gateway),
		}},
	}

	log.Printf("cmdAdd end")

	return types.PrintResult(result, conf.CNIVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	conf, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}
	log.Printf("cmdDel begin")

	pool, err := pool.NewKubeIPAMPool(&conf.IPAM)
	if err != nil {
		return err
	}

	alctr, err := allocator.NewRoundRobinAllocator()
	if err != nil {
		return err
	}

	return alctr.ReleaseBy(pool, args.ContainerID)
}
