package pool

import (
	"net"

	ippoolv1alpha1 "github.com/jbliao/kubeipam/api/v1alpha1"
)

// Address ...
type Address interface {
	Allocated() bool
	String() string
	NetIP() net.IP
}

// Pool is used for allocator
type Pool interface {
	GetAddresses() ([]Address, error)
	MarkAddressAllocated(Address, *ippoolv1alpha1.IPAllocation) error
	MarkAddressReleased(containerID string) error
}
