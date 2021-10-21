package netclient

import (
	networkv1 "wooshnet/apis/network/v1"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/extradhcpopts"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
)

type NetClient interface {
	Type() string
	GetSubnet(subnetId string) (*subnets.Subnet, error)
	ListSubnet(netId string) ([]subnets.Subnet, error)
	CreatePort(portName string, port *networkv1.Port) (*networkv1.Port, error)
	CreatePortWithExtraDHCPOpts(networkID, subnetID, portName string, dhcpOpt extradhcpopts.CreateExtraDHCPOpt) (*PortWithExtraDHCPOpts, error)
	DeletePort(portID string) error
}
