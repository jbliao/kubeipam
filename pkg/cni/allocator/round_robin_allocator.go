package allocator

import (
	"fmt"
	"net"

	"github.com/jbliao/kubeipam/pkg/cni"
)

type RoundRobinAllocator struct{}

// NewRoundRobinAllocator ...
func NewRoundRobinAllocator() (*RoundRobinAllocator, error) {
	return &RoundRobinAllocator{}, nil
}

// Allocate TODO
func (a *RoundRobinAllocator) Allocate(pool cni.Pool, usedBy string) (net.IP, error) {
	firstIP, lastIP, err := pool.GetFirstAndLastAddress()
	if err != nil {
		return nil, err
	}
	for ; !firstIP.Equal(lastIP); firstIP = cni.IncreaseIP(firstIP) {
		if pool.CheckAddressAvailable(firstIP) {
			break
		}
	}
	if firstIP.Equal(lastIP) {
		return nil, fmt.Errorf("cannot allocate address")
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
