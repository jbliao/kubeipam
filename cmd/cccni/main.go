package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"

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

func setupLog(logFile string) *log.Logger {
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0664)
		if err == nil {
			log.SetOutput(f)
		}
	}
	return log.New(log.Writer(), "", log.Flags()|log.Lshortfile)
}

func cmdAdd(args *skel.CmdArgs) error {
	conf, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}
	logger := setupLog(conf.IPAM.LogFile)
	logger.Printf("cmdAdd begin")

	pool, err := pool.NewKubeIPAMPool(&conf.IPAM, logger)
	if err != nil {
		log.Println(err)
		return err
	}

	alctr, err := allocator.NewRoundRobinAllocator(logger)
	if err != nil {
		log.Println(err)
		return err
	}

	ip, err := alctr.Allocate(pool, args.ContainerID)
	if err != nil {
		log.Println(err)
		return err
	}

	tmp := net.ParseIP(conf.IPAM.Mask)
	if tmp == nil {
		log.Println(err)
		return err
	}
	tmp = tmp.To4()

	result := &current.Result{
		IPs: []*current.IPConfig{{
			Version: "4",
			Address: net.IPNet{
				IP:   ip,
				Mask: net.IPv4Mask(tmp[0], tmp[1], tmp[2], tmp[3]),
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

	log.Printf("cmdAdd end %v", result)

	return types.PrintResult(result, conf.CNIVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	conf, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}
	logger := setupLog(conf.IPAM.LogFile)
	log.Printf("cmdDel begin")

	pool, err := pool.NewKubeIPAMPool(&conf.IPAM, logger)
	if err != nil {
		return err
	}

	alctr, err := allocator.NewRoundRobinAllocator(logger)
	if err != nil {
		return err
	}

	log.Printf("cmdDel end with err: %v", alctr.ReleaseBy(pool, args.ContainerID))
	return nil
}
