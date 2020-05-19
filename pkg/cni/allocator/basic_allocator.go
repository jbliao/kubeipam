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

// Allocate find the address in pool.addresses but not in pool.allocations
func (a *BasicAllocator) Allocate(pool cni.Pool, containerID string) (*ipaddr.IPAddress, error) {
	ipAddrLst, err := pool.GetAddresses()
	if err != nil {
		return nil, err
	}
	a.logger.Println("Loop to find allocable address")
	for _, ipAddr := range ipAddrLst {
		if _, allocated := ipAddr.Meta["allocated"]; !allocated {
			a.logger.Println("Found allocable address", ipAddr)
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

// Release just call pool.MarkAddressReleased which delete specific address from pool.allocations
func (a *BasicAllocator) Release(pool cni.Pool, addr *ipaddr.IPAddress, containerID string) error {
	a.logger.Println("Releasing address with ip&containerID", addr, containerID)
	return pool.MarkAddressReleased(addr, containerID)
}
