package driver

import (
	"fmt"
	"log"
	"net"

	"github.com/jbliao/kubeipam/api/v1alpha1"
)

type IpamAddressMark string

func (mark IpamAddressMark) String() string {
	return (string)(mark)
}

const (
	// Automated indicate that the address is created by k8s, not admin
	Automated IpamAddressMark = "k8s-automated"
	// Allocated indicate that the address is used by pod
	Allocated IpamAddressMark = "k8s-allocated"
)

type IpamAddress interface {
	Equal(net.IP) bool
	MarkedWith(IpamAddressMark) bool
	String() string
}

const ReserveAddressCount int = 1

// Driver for ipam syncing
type Driver interface {
	// GetAddresses get all address of this pool
	GetAddresses() ([]IpamAddress, error)

	// MarkAddressAllocated ensures that allocation is mark allocated in the ipam
	MarkAddressAllocated(addr IpamAddress, des string) error

	// MarkAddressReleased do the reverse
	MarkAddressReleased(addr IpamAddress) error

	// CreateAddress create an ip address in ipam system. Need to be thread-safe
	CreateAddress(count int) error

	// DeleteAddress delete an ip address in ipam system.
	DeleteAddress(addrs IpamAddress) error

	SetPoolID(string)
	SetLogger(*log.Logger)
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
			if !ipamAddr.MarkedWith(Allocated) && ipamAddr.MarkedWith(Automated) {
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
		var alct *v1alpha1.IPAllocation
		for _, alction := range spec.Allocations {
			ip := net.ParseIP(alction.Address)
			if ip == nil {
				err = fmt.Errorf("sync failed: cannot parse address %v",
					spec.Addresses)
				logger.Println(err)
				return err
			}
			alct = &alction
			if ipamAddr.Equal(ip) {
				toRelease = false
				break
			}
		}
		var err error
		if toRelease {
			err = d.MarkAddressReleased(ipamAddr)
		} else {
			err = d.MarkAddressAllocated(ipamAddr, alct.PodNamespace+"/"+alct.PodName)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
