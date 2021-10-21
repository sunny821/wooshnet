package netclient

import (
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
)

func (c *OvnClient) GetSubnet(subnetId string) (*subnets.Subnet, error) {
	// not implemented
	return nil, nil
}

func (c *OvnClient) ListSubnet(netId string) ([]subnets.Subnet, error) {
	// not implemented
	return nil, nil
}
