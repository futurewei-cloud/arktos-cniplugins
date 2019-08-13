package neutron

import (
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
)

// PortBindingDetail contains port binding detail
type PortBindingDetail struct {
	ports.Port
	portsbinding.PortsBindingExt
}

// Client encapsulates neutron interaction around port related 2.0 API
type Client struct {
	ServiceClient *gophercloud.ServiceClient
}

// New creates a Neutron Client
func New(user, password, tenant, identityURL string) (*Client, error) {
	var authOpts = gophercloud.AuthOptions{
		IdentityEndpoint: identityURL,
		Username:         user,
		Password:         password,
		TenantName:       tenant, //openstack project, a.k.a. VPC
		// todo: make DomainName configurable either in cni netconf file (per node) or from pod (per pod/user)
		DomainName: "default",
	}

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %v", err)
	}

	networkClient, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{
		Name:   "neutron",
		Region: "RegionOne", // todo: make it configurable in cni netconf file
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get network service client: %v", err)
	}

	return &Client{ServiceClient: networkClient}, nil
}

// GetPort gets detail of the neutron port by ID (not name)
func (c Client) GetPort(portID string) (*PortBindingDetail, error) {
	if portID == "" {
		return nil, fmt.Errorf("invalid portID: empty not allowed")
	}

	result := ports.Get(c.ServiceClient, portID)
	detail := PortBindingDetail{}
	if err := result.ExtractInto(&detail); err != nil {
		return nil, fmt.Errorf("failed to get port by %s: %v", portID, err)
	}

	return &detail, nil
}

// BindPort updates neutron port with host binding
func (c Client) BindPort(portID, hostID, devID string) (*PortBindingDetail, error) {
	if portID == "" {
		return nil, fmt.Errorf("invalid portID: empty not allowed")
	}

	if hostID == "" {
		return nil, fmt.Errorf("invalid hostID: empty not allowed")
	}

	if devID == "" {
		return nil, fmt.Errorf("invalid devID: empty not allowed")
	}

	deviceOwner := fmt.Sprintf("alktron:%s", devID)
	updateOpts := portsbinding.UpdateOptsExt{
		HostID: &hostID,
		UpdateOptsBuilder: ports.UpdateOpts{
			DeviceOwner: &deviceOwner,
		},
	}
	result := ports.Update(c.ServiceClient, portID, updateOpts)
	detail := PortBindingDetail{}
	if err := result.ExtractInto(&detail); err != nil {
		return nil, fmt.Errorf("failed to bind port %s to host %s: %v", portID, hostID, err)
	}

	return &detail, nil
}

// GetSubnet gets subnet detail from neutron service by subnet ID (not name)
func (c Client) GetSubnet(subnetID string) (*subnets.Subnet, error) {
	if subnetID == "" {
		return nil, fmt.Errorf("invalid subnetID: empty not allowed")
	}

	s, err := subnets.Get(c.ServiceClient, subnetID).Extract()
	if err != nil {
		return nil, fmt.Errorf("failed to get subnet %q: %v", subnetID, err)
	}

	return s, nil
}

// UnbindPort clears binding data and device owner property of the specified port in neutron service
func (c Client) UnbindPort(portID string) (*PortBindingDetail, error) {
	if portID == "" {
		return nil, fmt.Errorf("invalid portID: empty not allowed")
	}

	noHost := ""
	noOwner := ""
	updateOpts := portsbinding.UpdateOptsExt{
		HostID: &noHost,
		UpdateOptsBuilder: ports.UpdateOpts{
			DeviceOwner: &noOwner,
		},
	}
	result := ports.Update(c.ServiceClient, portID, updateOpts)
	detail := PortBindingDetail{}
	if err := result.ExtractInto(&detail); err != nil {
		return nil, fmt.Errorf("failed to unbind port %s: %v", portID, err)
	}

	return &detail, nil
}
