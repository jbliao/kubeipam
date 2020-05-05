package pool

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/jbliao/kubeipam/api/v1alpha1"
	"github.com/jbliao/kubeipam/pkg/cni"
	"github.com/jbliao/kubeipam/pkg/crd/clientset"
	"github.com/jbliao/kubeipam/pkg/ipaddr"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeIPAMPool implement Pool interface
type KubeIPAMPool struct {
	client *clientset.IPPoolClient
	config *cni.IPAMConf
	logger *log.Logger
	cache  *v1alpha1.IPPool
}

// NewKubeIPAMPool construct a KubeIPAMPool object
func NewKubeIPAMPool(ipamConf *cni.IPAMConf, logger *log.Logger) (*KubeIPAMPool, error) {
	if logger == nil {
		return nil, fmt.Errorf("nil logger in NewKubeIPAMPool")
	}

	config, err := clientcmd.BuildConfigFromFlags("", ipamConf.KubeConfigPath)
	if err != nil {
		logger.Println(err)
		return nil, err
	}
	client, err := clientset.NewForConfig(config, logger)

	if ipamConf.PoolNamespace == "" {
		//decide namespace from Kubectl Context if not given
		logger.Printf("PoolNamespace is empty, decide from context")
		var namespace string
		if cfg, err := clientcmd.LoadFromFile(ipamConf.KubeConfigPath); err != nil {
			logger.Println(err)
			return nil, err
		} else if ctx, ok := cfg.Contexts[cfg.CurrentContext]; ok && ctx != nil {
			namespace = ctx.Namespace
		} else {
			err := fmt.Errorf("k8s config: namespace not present in context")
			logger.Println(err)
			return nil, err
		}
		ipamConf.PoolNamespace = namespace
	}

	return &KubeIPAMPool{
		client: client,
		config: ipamConf,
		logger: logger,
	}, nil
}

func (p *KubeIPAMPool) ensureCache() error {
	var err error = nil
	if p.cache == nil {
		p.cache, err = p.client.GetIPPool(p.config.PoolNamespace, p.config.PoolName)
	}
	return err
}

func (p *KubeIPAMPool) updateWithCache() error {
	err := p.client.Update(context.Background(), p.cache)
	if err != nil {
		p.logger.Println(err)
	}
	return err
}

// GetFirstAndLastAddress get first and last address to this pool
func (p *KubeIPAMPool) GetFirstAndLastAddress() (*ipaddr.IPAddress, *ipaddr.IPAddress, error) {
	if err := p.ensureCache(); err != nil {
		return nil, nil, err
	}

	_, ipnet, err := net.ParseCIDR(p.cache.Spec.Range)
	if err != nil {
		p.logger.Println(err)
		return nil, nil, err
	}

	firstIP := ipaddr.NewIPAddress(ipnet.IP).IncreaseBy(1)
	lastIP := firstIP.GetBroadCastAddressWithMask(ipnet.Mask).IncreaseBy(-1)

	return firstIP, lastIP, nil
}

// CheckAddressAvailable check whether the given ip address is in pool's range,
// and not in allocations
func (p *KubeIPAMPool) CheckAddressAvailable(addr *ipaddr.IPAddress) (bool, error) {
	if err := p.ensureCache(); err != nil {
		return false, err
	}

	_, ipnet, err := net.ParseCIDR(p.cache.Spec.Range)
	if err != nil {
		p.logger.Println(err)
		return false, err
	}

	if !ipnet.Contains(addr.IP) {
		return false, nil
	}

	for _, alc := range p.cache.Spec.Allocations {
		alcaddr := net.ParseIP(alc.Address)
		if alcaddr == nil {
			return false, nil
		}
		if addr.Equal(alcaddr) {
			return false, nil
		}
	}
	return true, nil
}

// MarkAddressUsedBy append an IPAllocation object to allocations list. and call
// updateWithCache()
func (p *KubeIPAMPool) MarkAddressUsedBy(addr *ipaddr.IPAddress, usedBy string) error {
	ok, err := p.CheckAddressAvailable(addr)
	if !ok {
		err := fmt.Errorf("address not available: %v", err)
		p.logger.Println(err)
		return err
	}
	p.cache.Spec.Allocations = append(p.cache.Spec.Allocations,
		v1alpha1.IPAllocation{
			Address:     addr.String(),
			ContainerID: usedBy,
		},
	)
	return p.updateWithCache()
}

// MarkAddressReleasedBy remove an allocation indicated by ip, and call updateWithCache()
func (p *KubeIPAMPool) MarkAddressReleasedBy(addr *ipaddr.IPAddress, usedBy string) error {
	if err := p.ensureCache(); err != nil {
		return err
	}
	for idx, alc := range p.cache.Spec.Allocations {
		if addr.Equal(net.ParseIP(alc.Address)) || alc.ContainerID == usedBy {
			//remove
			p.logger.Printf("Found allocation to release: %v", alc)
			p.cache.Spec.Allocations = append(
				p.cache.Spec.Allocations[:idx],
				p.cache.Spec.Allocations[idx+1:]...,
			)
			return p.updateWithCache()
		}
	}
	err := fmt.Errorf("address %s or usedBy %s not found", addr.String(), usedBy)
	p.logger.Println(err)
	return err
}
