package netclient

import (
	networkv1 "wooshnet/apis/network/v1"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/extradhcpopts"
)

// // CreatePort will create a port on the specified subnet. An error will be
// // returned if the port could not be created.
func (c *OvnClient) CreatePort(portName string, port *networkv1.Port) (*networkv1.Port, error) {
	// not implemented
	return nil, nil
}

// CreatePortWithExtraDHCPOpts will create a port with DHCP options on the
// specified subnet. An error will be returned if the port could not be created.
func (c *OvnClient) CreatePortWithExtraDHCPOpts(networkID, subnetID, portName string, dhcpOpt extradhcpopts.CreateExtraDHCPOpt) (*PortWithExtraDHCPOpts, error) {
	// not implemented
	return nil, nil
}

// DeletePort will delete a port with a specified ID. A fatal error will
// occur if the delete was not successful. This works best when used as a
// deferred function.
func (c *OvnClient) DeletePort(portID string) error {
	// not implemented
	return nil
}
