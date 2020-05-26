package pool

import (
	"context"
	"fmt"
	"log"
	"net"

	ippoolv1alpha1 "github.com/jbliao/kubeipam/api/v1alpha1"
	"github.com/jbliao/kubeipam/pkg/cni"
	"github.com/jbliao/kubeipam/pkg/crd/clientset"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeIpamAddress impl Address interface
type KubeIpamAddress struct {
	net.IP
	allocated bool
}

// Allocated ...
func (a *KubeIpamAddress) Allocated() bool {
	return a.allocated
}

// NetIP return net.IP object
func (a *KubeIpamAddress) NetIP() net.IP {
	return a.IP
}

var _ Address = &KubeIpamAddress{}

// KubeIPAMPool implement Pool interface
type KubeIPAMPool struct {
	client *clientset.IPPoolClient
	config *cni.IPAMConf
	logger *log.Logger
	cache  *ippoolv1alpha1.IPPool
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

// GetAddresses get first and last address to this pool
func (p *KubeIPAMPool) GetAddresses() (ret []Address, err error) {

	if err = p.ensureCache(); err != nil {
		return
	}

	alctionSet := map[string]interface{}{}
	for _, alc := range p.cache.Spec.Allocations {
		alctionSet[alc.Address] = alc
	}

	for _, addr := range p.cache.Spec.Addresses {
		_, allocated := alctionSet[addr]
		ipa := &KubeIpamAddress{
			IP:        net.ParseIP(addr),
			allocated: allocated,
		}
		ret = append(ret, ipa)
	}

	return
}

// MarkAddressAllocated append an IPAllocation object to allocations list. and call
// updateWithCache()
func (p *KubeIPAMPool) MarkAddressAllocated(addr Address, info *ippoolv1alpha1.IPAllocation) error {
	if addr.Allocated() {
		err := fmt.Errorf("address allocated")
		p.logger.Println(err)
		return err
	}
	newObj := info.DeepCopy()
	newObj.Address = addr.String()
	p.cache.Spec.Allocations = append(p.cache.Spec.Allocations, *newObj)
	return p.updateWithCache()
}

func (p *KubeIPAMPool) deleteAllocationWithIndex(idx int) error {
	p.logger.Printf("Found allocation to release: %v", p.cache.Spec.Allocations[idx])
	p.cache.Spec.Allocations = append(
		p.cache.Spec.Allocations[:idx],
		p.cache.Spec.Allocations[idx+1:]...,
	)
	return p.updateWithCache()
}

// MarkAddressReleased remove an allocation indicated by ip, and call updateWithCache()
func (p *KubeIPAMPool) MarkAddressReleased(containerID string) error {
	if err := p.ensureCache(); err != nil {
		return err
	}
	p.logger.Println("Loop to find allocation to release")
	for idx, alc := range p.cache.Spec.Allocations {
		if alc.ContainerID == containerID {
			return p.deleteAllocationWithIndex(idx)
		}
	}
	p.logger.Printf("target not found %s", containerID)
	return nil
}

var _ Pool = &KubeIPAMPool{}
