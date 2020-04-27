package cni

import (
	"net"
)

// Pool is used for allocator
type Pool interface {
	GetFirstAndLastAddress() (net.IP, net.IP, error)
	CheckAddressAvailable(net.IP) (bool, error)
	MarkAddressUsedBy(net.IP, string) error
	MarkAddressReleasedBy(net.IP, string) error
}
