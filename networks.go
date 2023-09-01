package main

import (
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/pagination"
)

type Network struct {
	resource *networks.Network
	client   *gophercloud.ServiceClient
}

func (s Network) LastUpdated() time.Time {
	return s.resource.UpdatedAt
}

func (s Network) Delete() error {
	return networks.Delete(s.client, s.resource.ID).ExtractErr()
}

func (s Network) Type() string {
	return "network"
}

func (s Network) ID() string {
	return s.resource.ID
}

func (s Network) Name() string {
	return s.resource.Name
}

func (s Network) ClusterID() string {
	for _, tag := range s.resource.Tags {
		if value := strings.TrimPrefix(tag, "openshiftClusterID="); value != tag {
			return value
		}
	}
	return ""
}

func ListNetworks(client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := networks.List(client, nil).EachPage(func(page pagination.Page) (bool, error) {
			resources, err := networks.ExtractNetworks(page)
			for i := range resources {
				ch <- Network{
					resource: &resources[i],
					client:   client,
				}
			}
			return true, err
		}); err != nil {
			panic(err)
		}
	}()
	return ch
}
