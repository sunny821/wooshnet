package netclient

import (
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
)

func (c *OvnClient) GetNetwork(netId string) (*networks.Network, error) {
	// not implemented
	return nil, nil
}
