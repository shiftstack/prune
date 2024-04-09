package main

import (
	"context"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type Server struct {
	resource *servers.Server
	client   *gophercloud.ServiceClient
}

func (s Server) CreatedAt() time.Time {
	return s.resource.Created
}

func (s Server) Delete(ctx context.Context) error {
	return servers.Delete(ctx, s.client, s.resource.ID).ExtractErr()
}

func (s Server) Type() string {
	return "server"
}

func (s Server) ID() string {
	return s.resource.ID
}

func (s Server) Name() string {
	return s.resource.Name
}

func (s Server) Tags() []string {
	if s.resource.Tags != nil {
		return *s.resource.Tags
	}
	return nil
}

func (s Server) ClusterID() string {
	return s.resource.Metadata["openshiftClusterID"]
}

func ListServers(ctx context.Context, client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := servers.List(client, nil).EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
			resources, err := servers.ExtractServers(page)
			for i := range resources {
				ch <- &Server{
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
