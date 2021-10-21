/*
Copyright 2021 xiayuhai.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

// IP is a sub-struct that represents an individual IP.
type IP struct {
	SubnetID  string `json:"subnet_id"`
	IPAddress string `json:"ip_address,omitempty"`
	Gateway   string `json:"gateway,omitempty"`
	Cidr      string `json:"cidr,omitempty"`
}

// AddressPair contains the IP Address and the MAC address.
type AddressPair struct {
	IPAddress  string `json:"ip_address,omitempty"`
	MACAddress string `json:"mac_address,omitempty"`
}

// Port represents a Neutron port. See package documentation for a top-level
// description of what this is.
type Port struct {
	// UUID for the port.
	ID string `json:"id,omitempty"`

	// Network that this port is associated with.
	NetworkID string `json:"network_id,omitempty"`

	// Human-readable name for the port. Might not be unique.
	Name string `json:"name,omitempty"`

	// Describes the port.
	Description string `json:"description,omitempty"`

	// Administrative state of port. If false (down), port does not forward
	// packets.
	AdminStateUp bool `json:"admin_state_up,omitempty"`

	// Indicates whether network is currently operational. Possible values include
	// `ACTIVE', `DOWN', `BUILD', or `ERROR'. Plug-ins might define additional
	// values.
	Status string `json:"status,omitempty"`

	// Mac address to use on this port.
	MACAddress string `json:"mac_address,omitempty"`

	// Specifies IP addresses for the port thus associating the port itself with
	// the subnets where the IP addresses are picked from
	FixedIPs []IP `json:"fixed_ips,omitempty"`

	// TenantID is the project owner of the port.
	TenantID string `json:"tenant_id,omitempty"`

	// ProjectID is the project owner of the port.
	ProjectID string `json:"project_id,omitempty"`

	// Identifies the entity (e.g.: dhcp agent) using this port.
	DeviceOwner string `json:"device_owner,omitempty"`

	// Specifies the IDs of any security groups associated with a port.
	SecurityGroups []string `json:"security_groups,omitempty"`

	// Identifies the device (e.g., virtual server) using this port.
	DeviceID string `json:"device_id,omitempty"`

	// Identifies the list of IP addresses the port will recognize/accept
	AllowedAddressPairs []AddressPair `json:"allowed_address_pairs,omitempty"`

	// Tags optionally set via extensions/attributestags
	Tags []string `json:"tags,omitempty"`

	// QoSPolicyID represents an associated QoS policy.
	QoSPolicyID string `json:"qos_policy_id,omitempty"`
}

type PortStatus struct {
	Port        `json:",omitempty"`
	Index       int    `json:"index,omitempty"`     // 第index个设备
	PortID      string `json:"portId,omitempty"`    // neutron/ovn port唯一标识
	PortReady   bool   `json:"PortReady"`           // neutron/ovn port是否已经分配
	IfName      string `json:"ifname,omitempty"`    // CNI输入参数的设备名称
	NicName     string `json:"nicname,omitempty"`   // 移入netns的设备名称
	IfaceID     string `json:"ifaceid,omitempty"`   // 绑定ovs的设备名称
	Interface   string `json:"interface,omitempty"` // 实际修改后的设备名称
	DeviceReady bool   `json:"deviceReady"`         // 设备是否绑定ovs
}
