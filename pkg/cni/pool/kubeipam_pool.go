package pool

import (
	"context"
	"fmt"
	"net"

	"github.com/jbliao/kubeipam/api/v1alpha1"
	"github.com/jbliao/kubeipam/pkg/cni"
	"github.com/jbliao/kubeipam/pkg/crd/clientset"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeIPAMPool implement Pool interface
type KubeIPAMPool struct {
	client *clientset.IPPoolClient
	config *cni.IPAMConf
	cache  *v1alpha1.IPPool
}

// NewKubeIPAMPool construct a KubeIPAMPool object
func NewKubeIPAMPool(ipamConf *cni.IPAMConf) (*KubeIPAMPool, error) {
	config, err := clientcmd.BuildConfigFromFlags("", ipamConf.KubeConfigPath)
	if err != nil {
		return nil, err
	}
	client, err := clientset.NewForConfig(config)

	if ipamConf.PoolNamespace == "" {
		//decide namespace from Kubectl Context if not given
		var namespace string
		if cfg, err := clientcmd.LoadFromFile(ipamConf.KubeConfigPath); err != nil {
			return nil, err
		} else if ctx, ok := cfg.Contexts[cfg.CurrentContext]; ok && ctx != nil {
			namespace = ctx.Namespace
		} else {
			return nil, fmt.Errorf("k8s config: namespace not present in context")
		}
		ipamConf.PoolNamespace = namespace
	}

	return &KubeIPAMPool{
		client: client,
		config: ipamConf,
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
	return p.client.Update(context.Background(), p.cache)
}

// GetFirstAndLastAddress get first and last address to this pool
func (p *KubeIPAMPool) GetFirstAndLastAddress() (net.IP, net.IP, error) {
	if err := p.ensureCache(); err != nil {
		return nil, nil, err
	}

	ip, ipnet, err := net.ParseCIDR(p.cache.Spec.Range)
	if err != nil {
		return nil, nil, err
	}

	var lastIP, firstIP net.IP
	firstIP = cni.IncreaseIP(ipnet.IP)
	if len(ipnet.Mask) == 4 { //ipv4
		lastIP = ip.To4()
	}

	for idx := range lastIP {
		lastIP[idx] |= ^ipnet.Mask[idx]
	}
	lastIP[len(lastIP)-1]--

	return firstIP, lastIP, nil
}

// CheckAddressAvailable check whether the given ip address is in pool's range,
// and not in allocations
func (p *KubeIPAMPool) CheckAddressAvailable(ip net.IP) (bool, error) {
	if err := p.ensureCache(); err != nil {
		return false, err
	}

	_, ipnet, err := net.ParseCIDR(p.cache.Spec.Range)
	if err != nil {
		return false, err
	}

	if !ipnet.Contains(ip) {
		return false, nil
	}

	for _, alc := range p.cache.Spec.Allocations {
		alcaddr := net.ParseIP(alc.Address)
		if alcaddr == nil {
			return false, nil
		}
		if ip.Equal(alcaddr) {
			return false, nil
		}
	}
	return true, nil
}

// MarkAddressUsedBy append an IPAllocation object to allocations list. and call
// updateWithCache()
func (p *KubeIPAMPool) MarkAddressUsedBy(ip net.IP, usedBy string) error {
	ok, err := p.CheckAddressAvailable(ip)
	if !ok {
		return fmt.Errorf("address not available: %v", err)
	}
	p.cache.Spec.Allocations = append(p.cache.Spec.Allocations,
		v1alpha1.IPAllocation{
			Address:     ip.String(),
			ContainerID: usedBy,
		},
	)
	return p.updateWithCache()
}

// MarkAddressReleasedBy remove an allocation indicated by ip, and call updateWithCache()
func (p *KubeIPAMPool) MarkAddressReleasedBy(ip net.IP, usedBy string) error {
	if err := p.ensureCache(); err != nil {
		return err
	}
	for idx, alc := range p.cache.Spec.Allocations {
		if ip.Equal(net.ParseIP(alc.Address)) || alc.ContainerID == usedBy {
			//remove
			p.cache.Spec.Allocations = append(
				p.cache.Spec.Allocations[:idx],
				p.cache.Spec.Allocations[idx+1:]...,
			)
			return p.updateWithCache()
		}
	}
	return fmt.Errorf("address %v or usedBy %s not found", ip, usedBy)
}
