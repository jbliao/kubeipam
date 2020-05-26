package ipaddr

import (
	"fmt"
	"net"
	"testing"
)

func TestIPAddressIncrease(t *testing.T) {
	ip1 := NewIPAddress(net.ParseIP("10.1.1.2"))

	testCases := []struct {
		increase   int
		expectedIP string
	}{
		{50, "10.1.1.52"},
		{253, "10.1.1.255"},
		{254, "10.1.2.0"},
		{255, "10.1.2.1"},
		{65536, "10.2.1.2"},
		{-1, "10.1.1.1"},
		{-3, "10.1.0.255"},
		{-259, "10.0.255.255"},
		{-65535, "10.0.1.3"},
	}

	for _, tc := range testCases {
		if !ip1.IncreaseBy(tc.increase).Equal(net.ParseIP(tc.expectedIP)) {
			fmt.Printf("%v %v %v", ip1, tc.increase, tc.expectedIP)
			t.Fail()
		}
	}
}

func TestIPAddressLess(t *testing.T) {
	ip1 := NewIPAddress(net.ParseIP("10.1.1.2"))
	ip2 := NewIPAddress(net.ParseIP("10.1.1.2"))
	ip3 := NewIPAddress(net.ParseIP("192.168.254.158"))
	if ip1.LessThan(ip2) || ip2.LessThan(ip1) {
		t.Fail()
	}
	if !ip1.LessThan(ip3) || ip3.LessThan(ip1) {
		t.Fail()
	}
	if !ip2.LessThan(ip3) || ip3.LessThan(ip2) {
		t.Fail()
	}
}

func TestBroadCast(t *testing.T) {
	ip1 := NewIPAddress(net.ParseIP("10.1.1.1"))
	if !ip1.GetBroadCastAddressWithMask(net.CIDRMask(25, 32)).
		Equal(net.ParseIP("10.1.1.127")) {
		t.Fail()
	}
}
