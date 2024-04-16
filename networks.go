package main

import (
	"context"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type Network struct {
	resource *networks.Network
	client   *gophercloud.ServiceClient
}

func (s Network) CreatedAt() time.Time {
	return s.resource.CreatedAt
}

func (s Network) Delete(ctx context.Context) error {
	return networks.Delete(ctx, s.client, s.resource.ID).ExtractErr()
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

func (s Network) Tags() []string {
	return s.resource.Tags
}

func (s Network) ClusterID() string {
	for _, tag := range s.resource.Tags {
		if value := strings.TrimPrefix(tag, "openshiftClusterID="); value != tag {
			return value
		}
	}
	return ""
}

func ListNetworks(ctx context.Context, client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := networks.List(client, nil).EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
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
