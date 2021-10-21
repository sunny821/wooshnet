package cniapi

import (
	"encoding/json"
	"fmt"
	"strings"

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
	// Pod annotation for wooshnet ips, ex: '[{"subnet_id":"aaa"},{"subnet_id":"bbb", "ip":"192.168.10.10"},{"subnet_id":"ccc", "ip":"192.168.10.10", "internal":true}]'
	WooshNetProjectID = "projectId"
	WooshNetNetID     = "netId"
	WooshNetSubnetID  = "subnetId"
	WooshNetIPAddress = "ipAddress"
	WooshPortsAnnot   = "wooshnet/ports"
	WooshPortStatus   = "wooshnet/portstatus"
	WooshPortName     = "wooshnet/wooshport"
	WooshNetType      = "wooshcni"
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

// parse extra args i.e. FOO=BAR;ABC=123
func parseExtraArgs(args string) (map[string]string, error) {
	m := make(map[string]string)
	if len(args) == 0 {
		return m, nil
	}

	items := strings.Split(args, ";")
	for _, item := range items {
		kv := strings.Split(item, "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("CNI_ARGS invalid key/value pair: %s", kv)
		}
		m[kv[0]] = kv[1]
	}
	return m, nil
}
