package network

import (
	"encoding/json"
	"fmt"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/version"

	networkv1 "wooshnet/apis/network/v1"
)

const (
	// Pod annotation for multus default network-attachment-definition
	MultusDefaultNetworkAttachmentAnnot = "v1.multus-cni.io/default-network"
	// Pod annotation for network-attachment-definition
	NetworkAttachmentAnnot = "k8s.v1.cni.cncf.io/networks"
	// Pod annotation for network status
	NetworkStatusAnnot = "k8s.v1.cni.cncf.io/network-status"
	// Old Pod annotation for network status (which is used before but it will be obsolated)
	OldNetworkStatusAnnot = "k8s.v1.cni.cncf.io/networks-status"
	// Pod annotation for wooshnet ips, ex: '[{"fixed_ips":{"subnet_id":"aaa"}},{"fixed_ips":{"subnet_id":"bbb", "ip_address":"192.168.10.10"}},{"fixed_ips":{"subnet_id":"ccc", "ip_address":"192.168.20.11"},"security_groups":["xxx","yyy"], "qos_policy_id":"xxxxid"}]'
	WooshPortsAnnot        = "wooshnet/ports"
	WooshPortStatus        = "wooshnet/portstatus"
	WooshPortName          = "wooshnet/wooshport"
	WooshNetType           = "wooshcni"
	WooshNetProjectID      = "projectId"
	WooshNetNetID          = "netId"
	WooshNetSecurityGroups = "securityGroups"
	WooshNetSubnetID       = "subnetId"
	WooshNetIPAddress      = "ipAddress"
	WooshNetIFNamePrefix   = "eth"
	MultusPrefix           = "net"
)

type NetConf struct {
	types.NetConf
	Nested    bool             `json:"nested,omitempty"`
	WooshPort string           `json:"wooshPort,omitempty"`
	Ports     []networkv1.Port `json:"ports,omitempty"`
}

type NetConfList struct {
	CNIVersion string `json:"cniVersion,omitempty"`

	Name         string    `json:"name,omitempty"`
	DisableCheck bool      `json:"disableCheck,omitempty"`
	Plugins      []NetConf `json:"plugins,omitempty"`
}

type MultusConf struct {
	types.NetConf
	KubeConfig string        `json:"kubeconfig,omitempty"`
	Delegates  []NetConfList `json:"delegates,omitempty"`
}

func LoadNetConf(content []byte) (*NetConf, error) {
	n := &NetConf{}
	if err := json.Unmarshal(content, n); err != nil {
		return nil, fmt.Errorf("failed to load netconf: %v %q", err, string(content))
	}
	if err := version.ParsePrevResult(&n.NetConf); err != nil {
		return nil, err
	}
	return n, nil
}
