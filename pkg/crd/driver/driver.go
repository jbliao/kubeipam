package driver

import (
	"log"
	"net"

	mapset "github.com/deckarep/golang-set"
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
	alctedlst, err := d.GetAllocatedList(poolName)
	if err != nil {
		return err
	}

	// alctedset is the address set read from driver
	alctedset := mapset.NewSet()
	for _, alcted := range alctedlst {
		ip, _, err := net.ParseCIDR(alcted)
		if err != nil {
			logger.Println(err)
			return err
		}
		logger.Printf("alctedset add \"%v\"", ip.String())
		alctedset.Add(ip.String())
	}

	// alcionset is the address set read from kubernetes
	alcionset := mapset.NewSet()
	for _, alction := range spec.Allocations {
		alcionset.Add(alction.Address)
		logger.Printf("alcionset add \"%v\"", alction.Address)
		if !alctedset.Contains(alction.Address) {
			if err := d.CreateAllocated(poolName, &alction); err != nil {
				return err
			}
		}
	}

	for _, alcted := range alctedlst {
		ip, _, err := net.ParseCIDR(alcted)
		if err != nil {
			logger.Println(err)
			return err
		}
		if !alcionset.Contains(ip.String()) {
			logger.Printf("Deleting %v", alcted)
			d.DeleteAllocated(poolName, alcted)
		}
	}
	return nil
}
