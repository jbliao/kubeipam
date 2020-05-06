package cni

import (
	"github.com/jbliao/kubeipam/pkg/ipaddr"
)

// Pool is used for allocator
type Pool interface {
	GetAddresses() ([]*ipaddr.IPAddress, error)
	MarkAddressAllocated(*ipaddr.IPAddress, string) error
	MarkAddressReleased(*ipaddr.IPAddress, string) error
}
