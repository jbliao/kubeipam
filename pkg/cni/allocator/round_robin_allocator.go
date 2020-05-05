package allocator

import (
	"fmt"
	"log"

	"github.com/jbliao/kubeipam/pkg/cni"
	"github.com/jbliao/kubeipam/pkg/ipaddr"
)

// RoundRobinAllocator allocate with first available address
type RoundRobinAllocator struct {
	logger *log.Logger
}

// NewRoundRobinAllocator ...
func NewRoundRobinAllocator(logger *log.Logger) (*RoundRobinAllocator, error) {
	if logger == nil {
		return nil, fmt.Errorf("nil logger in NewRoundRobinAllocator")
	}
	return &RoundRobinAllocator{logger: logger}, nil
}

// Allocate TODO
func (a *RoundRobinAllocator) Allocate(pool cni.Pool, usedBy string) (*ipaddr.IPAddress, error) {
	firstAddr, lastAddr, err := pool.GetFirstAndLastAddress()
	if err != nil {
		return nil, err
	}
	a.logger.Printf("Allocate: got firstIP and lastIP %v %v", firstAddr, lastAddr)
	for ; !firstAddr.Equal(lastAddr.IP); firstAddr = firstAddr.IncreaseBy(1) {
		ok, err := pool.CheckAddressAvailable(firstAddr)
		if err != nil {
			return nil, err
		}
		if ok {
			break
		}
	}
	if firstAddr.Equal(lastAddr.IP) {
		err := fmt.Errorf("cannot allocate address %v %v", firstAddr, lastAddr)
		a.logger.Println(err)
		return nil, err
	}

	if err := pool.MarkAddressUsedBy(firstAddr, usedBy); err != nil {
		return nil, err
	}
	return firstAddr, nil
}

// Release TODO
func (a *RoundRobinAllocator) Release(pool cni.Pool, addr *ipaddr.IPAddress) error {
	return pool.MarkAddressReleasedBy(addr, "")
}

// ReleaseBy TODO
func (a *RoundRobinAllocator) ReleaseBy(pool cni.Pool, user string) error {
	return pool.MarkAddressReleasedBy(nil, user)
}
