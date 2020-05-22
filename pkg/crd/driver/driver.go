package driver

import (
	"fmt"
	"log"
	"net"

	"github.com/jbliao/kubeipam/api/v1alpha1"
)

type IpamAddress interface {
	Equal(net.IP) bool
	MarkedWith(string) bool
	String() string
}

const ReserveAddressCount int = 1

// Driver for ipam syncing
type Driver interface {
	// GetAddresses get all address of this pool
	GetAddresses() ([]IpamAddress, error)

	// MarkAddressAllocated ensures that allocation is mark allocated in the ipam
	MarkAddressAllocated(addr IpamAddress) error

	// MarkAddressReleased do the reverse
	MarkAddressReleased(addr IpamAddress) error

	CreateAddress(count int) error

	DeleteAddress(addrs IpamAddress) error
}

// Sync sync the allocations in spec with the pool identified by spec.Network
// TODO: rewrite the logic for more efficiency
func Sync(d Driver, spec *v1alpha1.IPPoolSpec, logger *log.Logger) error {

	logger.Println("Sync start")
	specAddressListSize := len(spec.Addresses)
	specAllocationListSize := len(spec.Allocations)
	sizeDiff := specAddressListSize - specAllocationListSize - ReserveAddressCount
	logger.Printf("address count=%d, allocation count=%d, reserve count=%d",
		specAddressListSize, specAllocationListSize, ReserveAddressCount)

	if sizeDiff < 0 {
		// need more address
		logger.Printf("need %d more address. creating...", -sizeDiff)
		if err := d.CreateAddress(-sizeDiff); err != nil {
			return err
		}
	}

	ipamAddrLst, err := d.GetAddresses()
	if err != nil {
		return err
	}

	if sizeDiff > 0 {
		logger.Println("too many address. deleting unallocated address...")
		tmpList := []IpamAddress{}
		for _, ipamAddr := range ipamAddrLst {
			if !ipamAddr.MarkedWith("k8s-allocated") {
				if err = d.DeleteAddress(ipamAddr); err != nil {
					return err
				}
				sizeDiff--
			}
			tmpList = append(tmpList, ipamAddr)
			if sizeDiff <= 0 {
				break
			}
		}
		ipamAddrLst = tmpList
	}

	// Sync addresses
	// Every addresses in driver is force sync to ippool now.
	logger.Println("Copying IpamAddr to AddressList")
	spec.Addresses = []string{}
	for _, ipamAddr := range ipamAddrLst {
		spec.Addresses = append(spec.Addresses, ipamAddr.String())
	}

	// Sync allocations
	// Every allocations in ippool is force sync to driver now.
	logger.Println("Mark allocation addresses alocated.")
	for _, ipamAddr := range ipamAddrLst {
		var toRelease bool = true
		for _, alction := range spec.Allocations {
			ip := net.ParseIP(alction.Address)
			if ip == nil {
				err = fmt.Errorf("sync failed: cannot parse address %v", spec.Addresses)
				logger.Println(err)
				return err
			}
			if ipamAddr.Equal(ip) {
				toRelease = false
				break
			}
		}
		var err error
		if toRelease {
			err = d.MarkAddressReleased(ipamAddr)
		} else {
			err = d.MarkAddressAllocated(ipamAddr)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
