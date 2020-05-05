package cni

import "github.com/jbliao/kubeipam/pkg/ipaddr"

// Pool is used for allocator
type Pool interface {
	GetFirstAndLastAddress() (*ipaddr.IPAddress, *ipaddr.IPAddress, error)
	CheckAddressAvailable(*ipaddr.IPAddress) (bool, error)
	MarkAddressUsedBy(*ipaddr.IPAddress, string) error
	MarkAddressReleasedBy(*ipaddr.IPAddress, string) error
}
