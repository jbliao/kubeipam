package allocator

import (
	"fmt"
	"log"

	ippoolv1alpha1 "github.com/jbliao/kubeipam/api/v1alpha1"
	"github.com/jbliao/kubeipam/pkg/cni/pool"
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
func (a *BasicAllocator) Allocate(pool pool.Pool, info *ippoolv1alpha1.IPAllocation) (pool.Address, error) {
	ipAddrLst, err := pool.GetAddresses()
	if err != nil {
		return nil, err
	}
	a.logger.Println("Loop to find allocable address")
	for _, ipAddr := range ipAddrLst {
		if !ipAddr.Allocated() {
			a.logger.Println("Found allocable address", ipAddr)
			if err := pool.MarkAddressAllocated(ipAddr, info); err != nil {
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
func (a *BasicAllocator) Release(pool pool.Pool, containerID string) error {
	a.logger.Printf("Releasing address with target %s", containerID)
	return pool.MarkAddressReleased(containerID)
}
