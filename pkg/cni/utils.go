package cni

import "net"

func IncreaseIP(ip net.IP) net.IP {
	length := len(ip)
	retIP := make([]byte, length)
	copy(retIP, ip)
	retIP[length-1]++
	for idx := length - 1; idx > 0; idx-- {
		if retIP[idx] != 0 {
			return retIP
		}
		retIP[idx-1]++
	}
	return retIP
}
