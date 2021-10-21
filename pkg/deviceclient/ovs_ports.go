package deviceclient

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	networkv1 "wooshnet/apis/network/v1"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
)

func (c *OvsClient) ConvertPort(port interface{}) *ports.Port {
	buf, _ := json.Marshal(port)
	result := &ports.Port{}
	_ = json.Unmarshal(buf, result)
	return result
}

// // CreatePort will create a port on the specified subnet. An error will be
// // returned if the port could not be created.
func (c *OvsClient) CreatePort(portName string, port *networkv1.PortStatus) (*networkv1.PortStatus, error) {
	args := []string{
		"--if-exists", "del-port", portName,
		"--", "add-port", c.Bridge, portName,
		"--", "set", "Interface", portName,
		"type=internal",
		"external-ids:iface-id=" + port.ID,
		"external-ids:iface-status=active",
		"external-ids:attached-mac=" + port.MACAddress,
		// "external-ids:vm-uuid=" + port.Description,
	}
	_, err := c.ovsExec(args...)
	if err != nil {
		return nil, err
	}
	return port, nil
}

// CreatePortWithMultipleFixedIPs will create a port with two FixedIPs on the
// specified subnet. An error will be returned if the port could not be created.
func (c *OvsClient) CreatePortWithMultipleFixedIPs(networkID, portName, portDescription string, fixedIPs []networkv1.IP) (*networkv1.PortStatus, error) {
	// not implemented
	return nil, nil
}

// DeletePort will delete a port with a specified ID. A fatal error will
// occur if the delete was not successful. This works best when used as a
// deferred function.
func (c *OvsClient) DeletePort(portID string) error {
	args := []string{"--if-exists", "del-port", c.Bridge, portID}
	_, err := c.ovsExec(args...)
	if err != nil {
		return err
	}
	return nil
}

func (c *OvsClient) ovsExec(args ...string) (string, error) {
	cmdargs := []string{"--timeout=30"}
	cmdargs = append(cmdargs, args...)
	output, err := exec.Command("ovs-vsctl", cmdargs...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run 'ovs-vsctl %s': %v\n  %q", strings.Join(cmdargs, " "), err, output)
	}

	outStr := string(output)
	trimmed := strings.TrimSpace(outStr)
	// If output is a single line, strip the trailing newline
	if strings.Count(trimmed, "\n") == 0 {
		outStr = trimmed
	}
	return outStr, nil
}
