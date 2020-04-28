package allocator

import (
	"fmt"
	"log"
	"net"

	"github.com/jbliao/kubeipam/pkg/cni"
)

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
func (a *RoundRobinAllocator) Allocate(pool cni.Pool, usedBy string) (net.IP, error) {
	firstIP, lastIP, err := pool.GetFirstAndLastAddress()
	if err != nil {
		return nil, err
	}
	a.logger.Printf("Allocate: got firstIP and lastIP %v %v", firstIP, lastIP)
	for ; !firstIP.Equal(lastIP); firstIP = cni.IncreaseIP(firstIP) {
		ok, err := pool.CheckAddressAvailable(firstIP)
		if err != nil {
			return nil, err
		}
		if ok {
			break
		}
	}
	if firstIP.Equal(lastIP) {
		err := fmt.Errorf("cannot allocate address %v %v", firstIP, lastIP)
		a.logger.Println(err)
		return nil, err
	}

	if err := pool.MarkAddressUsedBy(firstIP, usedBy); err != nil {
		return nil, err
	}
	return firstIP, nil
}

// Release TODO
func (a *RoundRobinAllocator) Release(pool cni.Pool, ip net.IP) error {
	return pool.MarkAddressReleasedBy(ip, "")
}

// ReleaseBy TODO
func (a *RoundRobinAllocator) ReleaseBy(pool cni.Pool, user string) error {
	return pool.MarkAddressReleasedBy(nil, user)
}
