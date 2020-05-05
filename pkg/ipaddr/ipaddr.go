package ipaddr

import (
	"net"
)

// IPAddress is a wrapper of net.IP
type IPAddress struct {
	net.IP
	Meta map[string]interface{}
}

// NewIPAddress ..
func NewIPAddress(raw net.IP) *IPAddress {
	ipa := IPAddress{}
	ipa.IP = append(ipa.IP, raw...)
	return &ipa
}

func (ipa IPAddress) copy() *IPAddress {
	newIPA := IPAddress{}
	newIPA.IP = append(newIPA.IP, ipa.IP...)
	for key, value := range ipa.Meta {
		newIPA.Meta[key] = value
	}
	return &newIPA
}

// IncreaseBy return a new IPAddress object that increased by a number(0~255)
func (ipa IPAddress) IncreaseBy(num int) *IPAddress {
	retIP := ipa.copy()
	for idx := len(retIP.IP) - 1; idx > 0; idx-- {
		tnum := (num + int(retIP.IP[idx])) / 256
		tval := (num + int(retIP.IP[idx])) % 256
		if tval < 0 {
			tnum--
			tval += 256
		}
		num = tnum
		retIP.IP[idx] = byte(tval)
	}
	return retIP
}

// LessThan check this object is less then the compared one
func (ipa IPAddress) LessThan(comp *IPAddress) bool {
	for idx := range ipa.IP {
		if ipa.IP[idx] > comp.IP[idx] {
			return false
		} else if ipa.IP[idx] < comp.IP[idx] {
			return true
		}
	}
	return false
}

// GetBroadCastAddressWithMask return a new IPAddress obj with broadcast ip
// counted by ip and mask
func (ipa IPAddress) GetBroadCastAddressWithMask(mask net.IPMask) *IPAddress {
	bcip := ipa.copy()
	if len(mask) == 4 {
		ones, bits := mask.Size()
		mask = net.CIDRMask(ones+96, bits+96)
	}

	for idx := range mask {
		bcip.IP[idx] |= ^mask[idx]
	}

	return bcip
}
