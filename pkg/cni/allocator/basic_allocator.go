package allocator

import (
	"fmt"
	"log"

	"github.com/jbliao/kubeipam/pkg/cni"
	"github.com/jbliao/kubeipam/pkg/ipaddr"
)

// BasicAllocator allocate with first available address
type BasicAllocator struct {
	logger *log.Logger
}

// NewBasicAllocator ...
func NewBasicAllocator(logger *log.Logger) (*BasicAllocator, error) {
	if logger == nil {
		return nil, fmt.Errorf("nil logger in NewBasicAllocator")
	}
	return &BasicAllocator{logger: logger}, nil
}

// Allocate TODO
func (a *BasicAllocator) Allocate(pool cni.Pool, containerID string) (*ipaddr.IPAddress, error) {
	ipAddrLst, err := pool.GetAddresses()
	if err != nil {
		return nil, err
	}

	for _, ipAddr := range ipAddrLst {
		if !ipAddr.Meta["allocated"].(bool) {
			if err := pool.MarkAddressAllocated(ipAddr, containerID); err != nil {
				return nil, err
			}
			return ipAddr, nil
		}
	}
	err = fmt.Errorf("cannot allocate")
	a.logger.Println(err)
	return nil, err
}

// Release TODO
func (a *BasicAllocator) Release(pool cni.Pool, addr *ipaddr.IPAddress, containerID string) error {
	return pool.MarkAddressReleased(addr, containerID)
}
