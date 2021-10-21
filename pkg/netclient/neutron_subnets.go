package netclient

import (
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
)

func (c *NeutronClient) GetSubnet(subnetId string) (*subnets.Subnet, error) {
	return subnets.Get(c.Client, subnetId).Extract()
}

func (c *NeutronClient) ListSubnet(netId string) ([]subnets.Subnet, error) {
	listOpts := subnets.ListOpts{
		NetworkID: netId,
	}
	allPages, err := subnets.List(c.Client, listOpts).AllPages()
	if err != nil {
		return []subnets.Subnet{}, nil
	}
	return subnets.ExtractSubnets(allPages)
}
