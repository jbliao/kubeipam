package cni

import (
	"github.com/jbliao/kubeipam/pkg/ipaddr"
)

// Allocator define the allocator interface
type Allocator interface {
	Allocate() (*ipaddr.IPAddress, error)
	Release(Pool, *ipaddr.IPAddress) error
	ReleaseBy(Pool, string) error
}
