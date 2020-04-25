package driver

import (
	mapset "github.com/deckarep/golang-set"
	"github.com/jbliao/kubeipam/api/v1alpha1"
)

// Driver for ipam syncing
type Driver interface {
	// RangeToPoolName convert ippool's range to driver's pool name
	RangeToPoolName(r string) (string, error)

	// GetAllocatedList ...
	GetAllocatedList(poolName string) ([]string, error)

	// EnsureAllocated ensures that allocation is mark allocated in the ipam
	CreateAllocated(poolName string, alc *v1alpha1.IPAllocation) error

	// EnsureUnAllocated do the reverse
	DeleteAllocated(poolName string, address string) error
}

// Sync sync the allocations in spec with the pool identified by spec.Range
func Sync(d Driver, spec *v1alpha1.IPPoolSpec) error {
	poolName, err := d.RangeToPoolName(spec.Range)
	if err != nil {
		return err
	}
	alctedlst, err := d.GetAllocatedList(poolName)
	if err != nil {
		return err
	}
	alctedset := mapset.NewSet()
	for _, alcted := range alctedlst {
		alctedset.Add(alcted)
	}

	alcionset := mapset.NewSet()
	for _, alction := range spec.Allocations {
		alcionset.Add(alction.Address)
		if !alctedset.Contains(alction.Address) {
			if err := d.CreateAllocated(poolName, &alction); err != nil {
				return err
			}
		}
	}

	for _, alcted := range alctedlst {
		if !alcionset.Contains(alcted) {
			d.DeleteAllocated(poolName, alcted)
		}
	}
	return nil
}