package netclient

import (
	"encoding/json"
	"fmt"

	networkv1 "wooshnet/apis/network/v1"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/acceptance/tools"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/extradhcpopts"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/qos/policies"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
)

// PortWithExtraDHCPOpts represents a port with extra DHCP options configuration.
type PortWithExtraDHCPOpts struct {
	ports.Port
	extradhcpopts.ExtraDHCPOptsExt
}

func (c *NeutronClient) ConvertPort(port interface{}) *networkv1.Port {
	buf, _ := json.Marshal(port)
	result := &networkv1.Port{}
	_ = json.Unmarshal(buf, result)
	return result
}

// // CreatePort will create a port on the specified subnet. An error will be
// // returned if the port could not be created.
func (c *NeutronClient) CreatePort(portName string, port *networkv1.Port) (*networkv1.Port, error) {
	var createOpts ports.CreateOptsBuilder
	createOpts = ports.CreateOpts{
		TenantID:       port.TenantID,
		ProjectID:      port.ProjectID,
		NetworkID:      port.NetworkID,
		Name:           portName,
		Description:    port.Description,
		AdminStateUp:   gophercloud.Enabled,
		FixedIPs:       port.FixedIPs,
		SecurityGroups: &port.SecurityGroups,
	}
	newPort := &ports.Port{}
	var err error
	if port.QoSPolicyID != "" {
		createOpts = policies.PortCreateOptsExt{
			CreateOptsBuilder: createOpts,
			QoSPolicyID:       port.QoSPolicyID,
		}
	}
	err = ports.Create(c.Client, createOpts).ExtractInto(newPort)
	if err != nil {
		return nil, err
	}
	result := c.ConvertPort(&newPort)
	result.QoSPolicyID = port.QoSPolicyID

	if err = c.WaitForPortToCreate(newPort.ID); err != nil {
		return result, err
	}

	return result, nil
}

// CreatePortWithExtraDHCPOpts will create a port with DHCP options on the
// specified subnet. An error will be returned if the port could not be created.
func (c *NeutronClient) CreatePortWithExtraDHCPOpts(networkID, subnetID, portName string, dhcpOpt extradhcpopts.CreateExtraDHCPOpt) (*PortWithExtraDHCPOpts, error) {
	portCreateOpts := ports.CreateOpts{
		NetworkID:    networkID,
		Name:         portName,
		AdminStateUp: gophercloud.Enabled,
		FixedIPs:     []ports.IP{{SubnetID: subnetID}},
	}

	createOpts := extradhcpopts.CreateOptsExt{
		CreateOptsBuilder: portCreateOpts,
		ExtraDHCPOpts:     []extradhcpopts.CreateExtraDHCPOpt{dhcpOpt},
	}
	port := &PortWithExtraDHCPOpts{}

	err := ports.Create(c.Client, createOpts).ExtractInto(port)
	if err != nil {
		return nil, err
	}

	if err := c.WaitForPortToCreate(port.ID); err != nil {
		return nil, err
	}

	err = ports.Get(c.Client, port.ID).ExtractInto(port)
	if err != nil {
		return port, err
	}

	return port, nil
}

func (c *NeutronClient) WaitForPortToCreate(portID string) error {
	return tools.WaitFor(func() (bool, error) {
		p, err := ports.Get(c.Client, portID).Extract()
		if err != nil {
			return false, err
		}

		if p.Status == "ACTIVE" || p.Status == "DOWN" {
			return true, nil
		}

		return false, nil
	})
}

// DeletePort will delete a port with a specified ID. A fatal error will
// occur if the delete was not successful. This works best when used as a
// deferred function.
func (c *NeutronClient) DeletePort(portID string) error {
	// t.Logf("Attempting to delete port: %s", portID)

	err := ports.Delete(c.Client, portID).ExtractErr()
	if err != nil {
		return fmt.Errorf("Unable to delete port %s: %v", portID, err)
	}

	// t.Logf("Deleted port: %s", portID)
	return nil
}
