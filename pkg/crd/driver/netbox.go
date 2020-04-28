package driver

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	runtimeclient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/jbliao/kubeipam/api/v1alpha1"
	"github.com/netbox-community/go-netbox/netbox"
	"github.com/netbox-community/go-netbox/netbox/client"
	"github.com/netbox-community/go-netbox/netbox/client/ipam"
	"github.com/netbox-community/go-netbox/netbox/models"
)

type NetboxDriver struct {
	Config NetboxDriverConfig
	Client *client.NetBox
	logger *log.Logger
}

type NetboxDriverConfig struct {
	Host   string `json:"host"`
	APIKey string `json:"apiKey"`
	Debug  bool   `json:"debug"`
}

func NewNetboxDriver(rawConfig string, logger *log.Logger) (*NetboxDriver, error) {
	ret := &NetboxDriver{}
	if logger == nil {
		return nil, fmt.Errorf("nil logger in NewNetboxDriver")
	}
	ret.logger = logger
	if err := json.Unmarshal([]byte(rawConfig), &ret.Config); err != nil {
		return nil, err
	}
	if ret.Config.Debug {
		logger.Println("Handle netbox in debug mode.")
		t := runtimeclient.New(ret.Config.Host, client.DefaultBasePath, client.DefaultSchemes)
		t.SetDebug(true)
		t.DefaultAuthentication =
			runtimeclient.APIKeyAuth(
				"Authorization",
				"header",
				fmt.Sprintf("Token %v", ret.Config.APIKey),
			)
		ret.Client = client.New(t, strfmt.Default)
	} else {
		ret.Client = netbox.NewNetboxWithAPIKey(ret.Config.Host, ret.Config.APIKey)
	}
	return ret, nil
}

// RangeToPoolName convert ippool's range to driver's pool name
// In Netbox they are the same value. So here just check the range fit cidr format
func (d *NetboxDriver) RangeToPoolName(rng string) (string, error) {
	_, _, err := net.ParseCIDR(rng)
	if err != nil {
		d.logger.Println(err)
		return "", err
	}
	return rng, nil
}

func (d *NetboxDriver) getAllocatedList(poolName string) ([]*models.IPAddress, error) {
	response, err := d.Client.Ipam.IpamIPAddressesList(
		ipam.NewIpamIPAddressesListParams().
			WithParent(&poolName), nil)
	if err != nil {
		d.logger.Println(err)
		return nil, err
	}
	return response.Payload.Results, nil
}

func (d *NetboxDriver) GetAllocatedList(poolName string) ([]string, error) {
	list, err := d.getAllocatedList(poolName)
	if err != nil {
		return nil, err
	}

	var ret []string
	for _, ip := range list {
		ret = append(ret, *ip.Address)
	}

	return ret, nil
}

func (d *NetboxDriver) CreateAllocated(poolName string, alc *v1alpha1.IPAllocation) error {

	// Do ip address in range check
	ip := net.ParseIP(alc.Address)
	if ip == nil {
		err := fmt.Errorf("Cannot parse address to ip in allocations")
		d.logger.Println(err)
		return err
	}
	_, pool, _ := net.ParseCIDR(poolName)
	if !pool.Contains(ip) {
		err := fmt.Errorf("IPAddress %s is not in range %s", alc, poolName)
		d.logger.Println(err)
		return err
	}

	addr := (&net.IPNet{IP: ip, Mask: pool.Mask}).String()
	data := &models.WritableIPAddress{
		Address: &addr,
		Tags:    []string{},
	}
	response, err := d.Client.Ipam.IpamIPAddressesCreate(
		ipam.NewIpamIPAddressesCreateParams().WithData(data),
		nil,
	)
	d.logger.Printf("Netbox create ipaddress with response: %v -- err: %v", response, err)
	return err
}

func (d *NetboxDriver) DeleteAllocated(poolName string, address string) error {
	iplist, err := d.getAllocatedList(poolName)
	if err != nil {
		return err
	}

	var id *int64 = nil
	for _, ipaddr := range iplist {
		if address == *ipaddr.Address {
			id = &ipaddr.ID
		}
	}

	if id == nil {
		return nil
	}

	response, err := d.Client.Ipam.IpamIPAddressesDelete(
		ipam.NewIpamIPAddressesDeleteParams().WithID(*id),
		nil,
	)
	d.logger.Printf("Netbox delete ipaddress with response %v -- err: %v", response, err)

	return err
}
