package cni

import (
	"net"
)

type Allocator interface {
	Allocate() (net.IP, error)
	Release(net.IP) error
}
