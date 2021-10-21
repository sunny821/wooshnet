package deviceclient

import (
	networkv1 "wooshnet/apis/network/v1"
)

type DeviceClient interface {
	Type() string
	CreatePort(portName string, port *networkv1.PortStatus) (*networkv1.PortStatus, error)
	CreatePortWithMultipleFixedIPs(networkID, portName, portDescription string, fixedIPs []networkv1.IP) (*networkv1.PortStatus, error)
	DeletePort(portID string) error
}
