package driver

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	runtimeclient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/jbliao/kubeipam/pkg/ipaddr"
	"github.com/netbox-community/go-netbox/netbox"
	"github.com/netbox-community/go-netbox/netbox/client"
	"github.com/netbox-community/go-netbox/netbox/client/ipam"
	"github.com/netbox-community/go-netbox/netbox/models"
)

// NetboxDriver impl the Driver interface with netbox support
type NetboxDriver struct {
	Config NetboxDriverConfig
	Client *client.NetBox
	logger *log.Logger
}

// NetboxDriverConfig contains the connection info to a netbox service
type NetboxDriverConfig struct {
	Host    string `json:"host"`
	APIKey  string `json:"apiKey"`
	Debug   bool   `json:"debug"`
	PoolKey string `json:"poolKey"`
}

// NewNetboxDriver construct a NetboxDriver instance with config
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

// NetworkToPoolName convert ippool's network to driver's pool name
// In Netbox they are the same value. So here just check the range fit cidr format
func (d *NetboxDriver) NetworkToPoolName(rng string) (string, error) {
	_, _, err := net.ParseCIDR(rng)
	if err != nil {
		d.logger.Println(err)
		return "", err
	}
	return rng, nil
}

func (d *NetboxDriver) getAddresses(poolName string) ([]*models.IPAddress, error) {
	response, err := d.Client.Ipam.IpamIPAddressesList(
		ipam.NewIpamIPAddressesListParams().
			WithParent(&poolName), nil)
	if err != nil {
		d.logger.Println(err)
		return nil, err
	}
	return response.Payload.Results, nil
}

// GetAllocatedList get allocated ip in netbox
func (d *NetboxDriver) GetAllocatedList(poolName string) ([]*ipaddr.IPAddress, error) {
	list, err := d.getAddresses(poolName)
	if err != nil {
		return nil, err
	}

	var ret []*ipaddr.IPAddress
	for _, ip := range list {
		for _, tag := range ip.Tags {
			if tag == "k8s" {
				ipa := ipaddr.NewIPAddress(net.ParseIP(*ip.Address))
				ipa.Meta["tags"] = ip.Tags
				ipa.Meta["id"] = ip.ID
				ret = append(ret, ipa)
				break
			}
		}
	}

	return ret, nil
}

// MarkAddressAllocated create an ipaddress object in netbox
func (d *NetboxDriver) MarkAddressAllocated(poolName string, addr *ipaddr.IPAddress) error {

	_, pool, _ := net.ParseCIDR(poolName)
	if !pool.Contains(addr.IP) {
		err := fmt.Errorf("IPAddress %s is not in range %s", addr.IP, poolName)
		d.logger.Println(err)
		return err
	}

	data := &models.WritableIPAddress{
		ID:   addr.Meta["id"].(int64),
		Tags: append(addr.Meta["tags"].([]string), "k8s-allocated"),
	}
	response, err := d.Client.Ipam.IpamIPAddressesPartialUpdate(
		ipam.NewIpamIPAddressesPartialUpdateParams().WithData(data),
		nil,
	)
	d.logger.Printf("Netbox create ipaddress with response: %v -- err: %v", response, err)
	return err
}

// MarkAddressReleased delete an ipaddress object in netbox
func (d *NetboxDriver) MarkAddressReleased(poolName string, addr *ipaddr.IPAddress) error {
	iplist, err := d.getAddresses(poolName)
	if err != nil {
		return err
	}

	var obj *models.IPAddress = nil
	for _, ip := range iplist {
		if addr.String() == *ip.Address {
			obj = ip
		}
	}

	if obj == nil {
		err := fmt.Errorf("object not found")
		d.logger.Println(err)
		return err
	}

	for idx, tag := range obj.Tags {
		if tag == "k8s-allocated" {
			obj.Tags = append(obj.Tags[:idx], obj.Tags[idx+1:]...)
			break
		}
	}

	data := models.WritableIPAddress{
		ID:   obj.ID,
		Tags: obj.Tags,
	}

	response, err := d.Client.Ipam.IpamIPAddressesPartialUpdate(
		ipam.NewIpamIPAddressesPartialUpdateParams().WithData(&data),
		nil,
	)
	d.logger.Printf("Netbox update ipaddress with response %v -- err: %v", response, err)

	return err
}
