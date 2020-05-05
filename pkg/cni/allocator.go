package cni

import (
	"github.com/jbliao/kubeipam/pkg/ipaddr"
)

type Allocator interface {
	Allocate() (*ipaddr.IPAddress, error)
	Release(Pool, *ipaddr.IPAddress) error
	ReleaseBy(Pool, string) error
}
