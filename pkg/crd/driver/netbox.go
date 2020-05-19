package driver

import (
	"fmt"
	"log"
	"net"

	runtimeclient "github.com/go-openapi/runtime/client"
	"github.com/jbliao/kubeipam/pkg/ipaddr"
	"github.com/netbox-community/go-netbox/netbox"
	"github.com/netbox-community/go-netbox/netbox/client"
	"github.com/netbox-community/go-netbox/netbox/client/ipam"
	"github.com/netbox-community/go-netbox/netbox/models"
)

// TODO: replace the use of IPAddress.Meta

// NetboxDriver impl the Driver interface with netbox support
type NetboxDriver struct {
	Client *client.NetBox
	logger *log.Logger
	prefix string
}

// NetboxDriverConfig contains the connection info to a netbox service
type NetboxDriverConfig struct {
	Host   string `json:"host"`
	APIKey string `json:"apiKey"`
	Debug  bool   `json:"debug"`
	Prefix string `json:"prefix"`
}

// NewNetboxDriver construct a NetboxDriver instance with config
func NewNetboxDriver(config *NetboxDriverConfig, logger *log.Logger) (nd *NetboxDriver, err error) {
	if logger == nil {
		return nil, fmt.Errorf("nil logger in NewNetboxDriver")
	}

	if config.Prefix == "" {
		err = fmt.Errorf("empty prefix")
		return
	} else if _, _, err = net.ParseCIDR(config.Prefix); err != nil {
		// Prefix needs to satisfy cidr format
		logger.Println(err)
		return
	}

	nd = &NetboxDriver{
		logger: logger,
		prefix: config.Prefix,
	}

	nd.Client = netbox.NewNetboxWithAPIKey(config.Host, config.APIKey)
	if config.Debug {
		logger.Println("Handle netbox in debug mode.")
		nd.Client.Transport.(*runtimeclient.Runtime).SetDebug(true)
	}
	return
}

func (d *NetboxDriver) getAddresses() ([]*models.IPAddress, error) {
	response, err := d.Client.Ipam.IpamIPAddressesList(
		ipam.NewIpamIPAddressesListParams().
			WithParent(&d.prefix), nil)
	if err != nil {
		d.logger.Println(err)
		return nil, err
	}
	return response.Payload.Results, nil
}

// GetAddresses get ip in netbox which allocated by k8s (has tag "k8s")
func (d *NetboxDriver) GetAddresses() ([]*ipaddr.IPAddress, error) {
	list, err := d.getAddresses()
	if err != nil {
		return nil, err
	}

	var ret []*ipaddr.IPAddress
	for _, ip := range list {
		for _, tag := range ip.Tags {
			if tag == "k8s" {
				netip, _, err := net.ParseCIDR(*ip.Address)
				if err != nil {
					d.logger.Println(err)
					return nil, err
				}
				ipa := ipaddr.NewIPAddress(netip)
				ipa.Meta["tags"] = ip.Tags
				ipa.Meta["id"] = ip.ID
				ipa.Meta["origin"] = ip.Address
				ret = append(ret, ipa)
				break
			}
		}
	}

	return ret, nil
}

// MarkAddressAllocated add "k8s-allocated" tag of netbox ipaddress resource
func (d *NetboxDriver) MarkAddressAllocated(addr *ipaddr.IPAddress) error {

	_, pool, _ := net.ParseCIDR(d.prefix)
	if !pool.Contains(addr.IP) {
		err := fmt.Errorf("IPAddress %s is not in range %s", addr.IP, d.prefix)
		d.logger.Println(err)
		return err
	}

	data := &models.WritableIPAddress{
		ID:      addr.Meta["id"].(int64),
		Address: addr.Meta["origin"].(*string),
		Tags:    append(addr.Meta["tags"].([]string), "k8s-allocated"),
	}
	response, err := d.Client.Ipam.IpamIPAddressesPartialUpdate(
		ipam.NewIpamIPAddressesPartialUpdateParams().WithID(data.ID).WithData(data),
		nil,
	)
	d.logger.Printf("Netbox create ipaddress with response: %v -- err: %v", response, err)
	return err
}

// MarkAddressReleased remove "k8s-allocated" tag of netbox ipaddress resource
func (d *NetboxDriver) MarkAddressReleased(addr *ipaddr.IPAddress) error {

	tags := addr.Meta["tags"].([]string)
	for idx, tag := range tags {
		if tag == "k8s-allocated" {
			tags = append(tags[:idx], tags[idx+1:]...)
			break
		}
	}

	data := models.WritableIPAddress{
		ID:      addr.Meta["id"].(int64),
		Tags:    tags,
		Address: addr.Meta["origin"].(*string),
	}

	response, err := d.Client.Ipam.IpamIPAddressesPartialUpdate(
		ipam.NewIpamIPAddressesPartialUpdateParams().WithID(data.ID).WithData(&data),
		nil,
	)
	d.logger.Printf("Netbox update ipaddress with response %v -- err: %v", response, err)

	return err
}

// CreateAddress create addresses on ipam system and claim those will be used
// by k8s
func (d *NetboxDriver) CreateAddress(count int) (err error) {

	// count need greater than zero
	// TODO: consider this a warning, not error.
	if count < 0 {
		err = fmt.Errorf("count less than 0")
		d.logger.Println(err)
		return
	}

	// get id of the prefix that indecated by poolName(a prefix string)
	response, err := d.Client.Ipam.IpamPrefixesList(
		ipam.NewIpamPrefixesListParams().WithPrefix(&d.prefix), nil)
	if err != nil {
		d.logger.Println(err)
		return
	}

	if *response.Payload.Count != 1 {
		err = fmt.Errorf("cannot find or decide prefix %s: %v, %v",
			d.prefix,
			response.Payload,
			err)
		d.logger.Println(err)
		return
	}

	// create addresses
	prefixID := response.Payload.Results[0].ID
	for ; count > 0; count-- {
		var createResponse *ipam.IpamPrefixesAvailableIpsCreateCreated
		createResponse, err = d.Client.Ipam.IpamPrefixesAvailableIpsCreate(
			ipam.NewIpamPrefixesAvailableIpsCreateParams().
				WithID(prefixID).
				// data should be a WritableIPAddress object. this may be a bug of netbox
				WithData(&models.WritablePrefix{
					Tags: []string{"k8s"},
				}),
			nil,
		)
		d.logger.Printf("Netbox create ipaddress with response %v err %v", createResponse, err)
		if err != nil {
			return
		}
	}
	return
}

// DeleteAddress delete IPAddresses from netbox
func (d *NetboxDriver) DeleteAddress(addr *ipaddr.IPAddress) (err error) {
	id, ok := addr.Meta["id"].(int64)
	if ok {
		var response *ipam.IpamIPAddressesDeleteNoContent
		response, err = d.Client.Ipam.IpamIPAddressesDelete(
			ipam.NewIpamIPAddressesDeleteParams().WithID(id),
			nil,
		)
		d.logger.Printf("Netbox create ipaddress with response %v err %v", response, err)
	} else {
		err = fmt.Errorf("id not found in object meta")
		d.logger.Println(err)
	}
	return
}
