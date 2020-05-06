package driver

import (
	"fmt"
	"log"
	"net"

	"github.com/jbliao/kubeipam/api/v1alpha1"
	"github.com/jbliao/kubeipam/pkg/ipaddr"
)

// Driver for ipam syncing
type Driver interface {
	// NetworkToPoolName convert ippool's network to driver's pool name
	NetworkToPoolName(network string) (string, error)

	// GetAddresses get all address of this pool
	GetAddresses(poolName string) ([]*ipaddr.IPAddress, error)

	// MarkAddressAllocated ensures that allocation is mark allocated in the ipam
	MarkAddressAllocated(poolName string, addr *ipaddr.IPAddress) error

	// MarkAddressReleased do the reverse
	MarkAddressReleased(poolName string, addr *ipaddr.IPAddress) error
}

// Sync sync the allocations in spec with the pool identified by spec.Network
func Sync(d Driver, spec *v1alpha1.IPPoolSpec, logger *log.Logger) error {

	poolName, err := d.NetworkToPoolName(spec.Network)
	if err != nil {
		return err
	}

	ipamAddrLst, err := d.GetAddresses(poolName)
	if err != nil {
		return err
	}

	// Sync addresses
	// Every addresses in driver is force sync to ippool now.
	logger.Println("Syncing addresses")
	spec.Addresses = []string{}
	for _, ipamAddr := range ipamAddrLst {
		spec.Addresses = append(spec.Addresses, ipamAddr.String())
	}

	// Sync allocations
	// Every allocations in ippool is force sync to driver now.
	logger.Println("Syncing allocations")
	for _, ipamAddr := range ipamAddrLst {
		var toRelease bool = true
		for _, alction := range spec.Allocations {
			ip := net.ParseIP(alction.Address)
			if ip == nil {
				return fmt.Errorf("sync failed %v", spec.Addresses)
			}
			if ipamAddr.Equal(ip) {
				toRelease = false
				break
			}
		}
		var err error
		if toRelease {
			logger.Println("Releasing ", ipamAddr)
			err = d.MarkAddressReleased(poolName, ipamAddr)
		} else {
			logger.Println("Allocating ", ipamAddr)
			err = d.MarkAddressAllocated(poolName, ipamAddr)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
