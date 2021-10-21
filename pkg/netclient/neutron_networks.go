package netclient

import (
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
)

func (c *NeutronClient) GetNetwork(netId string) (*networks.Network, error) {
	return networks.Get(c.Client, netId).Extract()
}
