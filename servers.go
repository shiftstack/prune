package main

import (
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/pagination"
)

type Server struct {
	resource *servers.Server
	client   *gophercloud.ServiceClient
}

func (s Server) CreatedAt() time.Time {
	return s.resource.Created
}

func (s Server) Delete() error {
	return servers.Delete(s.client, s.resource.ID).ExtractErr()
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

func (s Server) ClusterID() string {
	return s.resource.Metadata["openshiftClusterID"]
}

func ListServers(client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := servers.List(client, nil).EachPage(func(page pagination.Page) (bool, error) {
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
