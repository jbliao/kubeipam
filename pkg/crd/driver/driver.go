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

	CreateAddress(poolName string, count int) error

	DeleteAddress(poolName string, addrs *ipaddr.IPAddress) error
}

// Sync sync the allocations in spec with the pool identified by spec.Network
func Sync(d Driver, spec *v1alpha1.IPPoolSpec, logger *log.Logger) error {

	poolName, err := d.NetworkToPoolName(spec.Network)
	if err != nil {
		return err
	}

	specAddressListSize := len(spec.Addresses)
	specAllocationListSize := len(spec.Allocations)
	sizeDiff := specAddressListSize - specAllocationListSize - 1

	if sizeDiff < 0 {
		// need more address
		logger.Println("need more address. creating")
		if err := d.CreateAddress(poolName, -sizeDiff); err != nil {
			return err
		}
	}

	ipamAddrLst, err := d.GetAddresses(poolName)
	if err != nil {
		return err
	}

	if sizeDiff > 0 {
		logger.Println("too many address. deleting...")
		for idx, ipamAddr := range ipamAddrLst {
			allocated := false
			for _, tag := range ipamAddr.Meta["tags"].([]string) {
				if tag == "k8s-allocated" {
					allocated = true
				}
			}
			if !allocated {
				if err = d.DeleteAddress(poolName, ipamAddr); err != nil {
					return err
				}
				ipamAddrLst = append(ipamAddrLst[:idx], ipamAddrLst[idx+1:]...)
				sizeDiff--
			}
			if sizeDiff == 0 {
				break
			}
		}
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
