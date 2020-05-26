package driver

import (
	"fmt"
	"log"
	"net"

	"github.com/go-openapi/runtime"
	runtimeclient "github.com/go-openapi/runtime/client"
	"github.com/netbox-community/go-netbox/netbox"
	"github.com/netbox-community/go-netbox/netbox/client"
	"github.com/netbox-community/go-netbox/netbox/client/ipam"
	"github.com/netbox-community/go-netbox/netbox/models"
)

// NetboxIPAddress ...
type NetboxIPAddress struct {
	net.IP
	tagset      map[string]interface{}
	origin      *models.IPAddress
	description string
}

// MarkedWith impl IpamAddress.MarkedWith with netbox tag feature
func (nba *NetboxIPAddress) MarkedWith(markStr string) bool {
	_, ok := nba.tagset[markStr]
	return ok
}

func (nba *NetboxIPAddress) tagsArray() (tags []string) {
	for tag := range nba.tagset {
		tags = append(tags, tag)
	}
	return
}

func (nba *NetboxIPAddress) removeTag(tag string) *NetboxIPAddress {
	delete(nba.tagset, tag)
	return nba
}

func (nba *NetboxIPAddress) addTag(tag string) *NetboxIPAddress {
	nba.tagset[tag] = struct{}{}
	return nba
}

func (nba *NetboxIPAddress) hasTag(tag string) bool {
	_, ok := nba.tagset[tag]
	return ok
}

// Make sure the NetboxIPAddress struct satisfy the IpamAddress interface
var _ IpamAddress = &NetboxIPAddress{}

// NetboxDriver impl the Driver interface with netbox support
type NetboxDriver struct {
	client *client.NetBox
	logger *log.Logger
	prefix string
	poolID string
}

// NetboxDriverConfig contains the connection info to a netbox service
type NetboxDriverConfig struct {
	Host   string `json:"host"`
	APIKey string `json:"apiKey"`
	Debug  bool   `json:"debug"`
	Prefix string `json:"prefix"`
}

// NewNetboxDriver construct a NetboxDriver instance with config
func NewNetboxDriver(config *NetboxDriverConfig) (nd *NetboxDriver, err error) {

	if config.Prefix == "" {
		err = fmt.Errorf("empty prefix")
		return
	} else if _, _, err = net.ParseCIDR(config.Prefix); err != nil {
		// Prefix needs to satisfy cidr format
		log.Println(err)
		return
	}

	nd = &NetboxDriver{
		prefix: config.Prefix,
		logger: log.New(log.Writer(), "Netbox", log.Flags()),
		client: netbox.NewNetboxWithAPIKey(config.Host, config.APIKey),
	}

	if config.Debug {
		log.Println("Handle netbox in debug mode.")
		nd.client.Transport.(*runtimeclient.Runtime).SetDebug(true)
	}
	return
}

func (d *NetboxDriver) getAddresses() ([]*models.IPAddress, error) {
	response, err := d.client.Ipam.IpamIPAddressesList(
		ipam.NewIpamIPAddressesListParams().
			WithParent(&d.prefix), nil)
	if err != nil {
		d.logger.Println(err)
		return nil, err
	}
	return response.Payload.Results, nil
}

func (d *NetboxDriver) poolIDTag() string {
	return fmt.Sprintf("k8s-pool-%s", d.poolID)
}

// GetAddresses get ip in netbox which allocated by k8s (has tag "k8s")
func (d *NetboxDriver) GetAddresses() (ret []IpamAddress, err error) {
	list, err := d.getAddresses()
	if err != nil {
		return
	}

	for _, modelAddr := range list {

		tagset := map[string]interface{}{}
		for _, tag := range modelAddr.Tags {
			tagset[tag] = struct{}{}
		}
		netip, _, err := net.ParseCIDR(*modelAddr.Address)
		if err != nil {
			d.logger.Println(err)
			return nil, err
		}
		ipa := &NetboxIPAddress{
			tagset: tagset,
			origin: modelAddr,
			IP:     netip,
		}

		if ipa.hasTag(d.poolIDTag()) {
			ret = append(ret, ipa)
		}
	}

	return
}

// MarkAddressAllocated add "k8s-allocated" tag of netbox ipaddress resource
func (d *NetboxDriver) MarkAddressAllocated(addr IpamAddress, des string) (err error) {

	netboxAddr, ok := addr.(*NetboxIPAddress)
	if !ok {
		err = fmt.Errorf("cannot assert addr to NetboxIPAddress")
		d.logger.Println(err)
		return
	}

	if netboxAddr.hasTag(Allocated) {
		return nil
	}

	_, pool, _ := net.ParseCIDR(d.prefix)
	if !pool.Contains(netboxAddr.IP) {
		err = fmt.Errorf("IPAddress %s is not in range %s",
			netboxAddr.IP, d.prefix)
		d.logger.Println(err)
		return
	}

	response, err := d.client.Ipam.IpamIPAddressesPartialUpdate(
		ipam.NewIpamIPAddressesPartialUpdateParams().
			WithID(netboxAddr.origin.ID).
			WithData(&models.WritableIPAddress{
				ID:          netboxAddr.origin.ID,
				Address:     netboxAddr.origin.Address,
				Tags:        netboxAddr.addTag(Allocated).tagsArray(),
				Description: des,
			}),
		nil,
	)

	d.logger.Printf("Netbox update ipaddress with response: %v -- err: %v",
		response, err)

	if err == nil {
		d.logger.Printf("Address %s marked allocated.", addr.String())
	}

	return
}

// MarkAddressReleased remove "k8s-allocated" tag of netbox ipaddress resource
// this function is not thread-safe
func (d *NetboxDriver) MarkAddressReleased(addr IpamAddress) (err error) {

	netboxAddr, ok := addr.(*NetboxIPAddress)
	if !ok {
		err = fmt.Errorf("cannot assert addr to NetboxIPAddress")
		d.logger.Println(err)
		return
	}

	if !netboxAddr.hasTag(Allocated) {
		return nil
	}

	response, err := d.client.Ipam.IpamIPAddressesPartialUpdate(
		ipam.NewIpamIPAddressesPartialUpdateParams().
			WithID(netboxAddr.origin.ID).
			WithData(&models.WritableIPAddress{
				ID:      netboxAddr.origin.ID,
				Tags:    netboxAddr.removeTag(Allocated).tagsArray(),
				Address: netboxAddr.origin.Address,
			}),
		nil,
	)
	d.logger.Printf("Netbox update ipaddress with response %v -- err: %v",
		response, err)

	if err == nil {
		d.logger.Printf("Address %s marked released.", addr.String())
	}

	return
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
	response, err := d.client.Ipam.IpamPrefixesList(
		ipam.NewIpamPrefixesListParams().WithPrefix(&d.prefix), nil)
	if err != nil {
		d.logger.Println(err)
		return
	}

	if *response.Payload.Count != 1 {
		err = fmt.Errorf("cannot find or decide prefix %s: %v, %v",
			d.prefix, response.Payload, err)
		d.logger.Println(err)
		return
	}

	// create addresses
	prefixID := response.Payload.Results[0].ID
	for ; count > 0; count-- {
		var createResponse *ipam.IpamPrefixesAvailableIpsCreateOK
		createResponse, err = d.client.Ipam.IpamPrefixesAvailableIpsCreate(
			ipam.NewIpamPrefixesAvailableIpsCreateParams().
				WithID(prefixID).
				// data should be a WritableIPAddress object. this may be a bug of go-netbox
				WithData(&models.WritablePrefix{
					Tags: []string{d.poolIDTag(), Automated},
				}),
			nil,
		)
		d.logger.Printf("Netbox create ipaddress with response %v err %v",
			createResponse, err)
		if err != nil {
			//TODO: Workaround for https://github.com/netbox-community/netbox/issues/4674
			if err.(*runtime.APIError).Code == 201 {
				d.logger.Printf("Netbox returned 201")
				d.logger.Println(err.(*runtime.APIError).Response)
				err = nil
				return
			}
			return
		}
		d.logger.Printf("Address %s created", createResponse.Payload[0].Address)
	}
	return
}

// DeleteAddress delete IPAddresses from netbox
func (d *NetboxDriver) DeleteAddress(addr IpamAddress) (err error) {
	netboxAddr, ok := addr.(*NetboxIPAddress)
	if !ok {
		return fmt.Errorf("cannot assert addr to NetboxIPAddress")
	}

	if !netboxAddr.hasTag(Automated) {
		err = fmt.Errorf("Cannot delete address which not auto created")
		d.logger.Println(err)
		return
	}

	if netboxAddr.origin != nil {
		var response *ipam.IpamIPAddressesDeleteNoContent
		response, err = d.client.Ipam.IpamIPAddressesDelete(
			ipam.NewIpamIPAddressesDeleteParams().
				WithID(netboxAddr.origin.ID),
			nil,
		)
		d.logger.Printf("Netbox delete ipaddress with response %v err %v",
			response, err)
		if err != nil {
			return
		}
		d.logger.Printf("Address %s deleted.", addr.String())
	} else {
		err = fmt.Errorf("id not found in object meta")
		d.logger.Println(err)
	}
	return
}

// SetPoolID ...
func (d *NetboxDriver) SetPoolID(poolID string) {
	d.poolID = poolID
}

// SetLogger ...
func (d *NetboxDriver) SetLogger(lgr *log.Logger) {
	if lgr != nil {
		d.logger = lgr
	}
}

var _ Driver = &NetboxDriver{}
