package netclient

import (
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/acceptance/clients"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	corev1 "k8s.io/api/core/v1"
)

type NeutronClient struct {
	Client *gophercloud.ServiceClient
}

func NewNeutronClient(config *corev1.ConfigMap) (*NeutronClient, error) {
	os.Setenv("OS_AUTH_URL", config.Data["OS_AUTH_URL"])
	os.Setenv("OS_USERNAME", config.Data["OS_USERNAME"])
	os.Setenv("OS_USERID", config.Data["OS_USERID"])
	os.Setenv("OS_PASSWORD", config.Data["OS_PASSWORD"])
	os.Setenv("OS_PASSCODE", config.Data["OS_PASSCODE"])
	os.Setenv("OS_TENANT_ID", config.Data["OS_TENANT_ID"])
	os.Setenv("OS_TENANT_NAME", config.Data["OS_TENANT_NAME"])
	os.Setenv("OS_DOMAIN_ID", config.Data["OS_DOMAIN_ID"])
	os.Setenv("OS_DOMAIN_NAME", config.Data["OS_DOMAIN_NAME"])

	client, err := clients.NewNetworkV2Client()
	if err != nil {
		return nil, err
	}
	return &NeutronClient{Client: client}, nil
}

func (c *NeutronClient) Type() string {
	return "neutron"
}

// This is duplicated from https://github.com/gophercloud/utils
// so that Gophercloud "core" doesn't have a dependency on the
// complementary utils repository.
func (c *NeutronClient) IDFromName(name string) (string, error) {
	count := 0
	id := ""

	listOpts := networks.ListOpts{
		Name: name,
	}

	pages, err := networks.List(c.Client, listOpts).AllPages()
	if err != nil {
		return "", err
	}

	all, err := networks.ExtractNetworks(pages)
	if err != nil {
		return "", err
	}

	for _, s := range all {
		if s.Name == name {
			count++
			id = s.ID
		}
	}

	switch count {
	case 0:
		return "", gophercloud.ErrResourceNotFound{Name: name, ResourceType: "network"}
	case 1:
		return id, nil
	default:
		return "", gophercloud.ErrMultipleResourcesFound{Name: name, Count: count, ResourceType: "network"}
	}
}
