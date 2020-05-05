package ipaddr

import (
	"net"
	"testing"
)

func TestIPAddressIncrease(t *testing.T) {
	ip1 := NewIPAddress(net.ParseIP("10.1.1.2"))

	if !ip1.IncreaseBy(50).Equal(net.ParseIP("10.1.1.52")) {
		t.Fail()
	}
	if !ip1.IncreaseBy(253).Equal(net.ParseIP("10.1.1.255")) {
		t.Fail()
	}
	if !ip1.IncreaseBy(254).Equal(net.ParseIP("10.1.2.0")) {
		t.Fail()
	}
	if !ip1.IncreaseBy(255).Equal(net.ParseIP("10.1.2.1")) {
		t.Fail()
	}
	if !ip1.IncreaseBy(65536).Equal(net.ParseIP("10.2.1.2")) {
		t.Fail()
	}

	if !ip1.IncreaseBy(-1).Equal(net.ParseIP("10.1.1.1")) {
		t.Fail()
	}
	if !ip1.IncreaseBy(-3).Equal(net.ParseIP("10.1.0.255")) {
		t.Fail()
	}
	if !ip1.IncreaseBy(-259).Equal(net.ParseIP("10.0.255.255")) {
		t.Fail()
	}
	if !ip1.IncreaseBy(-65535).Equal(net.ParseIP("10.0.1.3")) {
		t.Fail()
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
