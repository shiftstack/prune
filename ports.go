package main

import (
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/pagination"
)

type Port struct {
	resource *ports.Port
	client   *gophercloud.ServiceClient
}

func (s Port) LastUpdated() time.Time {
	return s.resource.UpdatedAt
}

func (s Port) Delete() error {
	return ports.Delete(s.client, s.resource.ID).ExtractErr()
}

func (s Port) Type() string {
	return "port"
}

func (s Port) ID() string {
	return s.resource.ID
}

func (s Port) Name() string {
	return s.resource.Name
}

func (s Port) ClusterID() string {
	for _, tag := range s.resource.Tags {
		if value := strings.TrimPrefix(tag, "openshiftClusterID="); value != tag {
			return value
		}
	}
	return ""
}

func ListPorts(client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := ports.List(client, nil).EachPage(func(page pagination.Page) (bool, error) {
			resources, err := ports.ExtractPorts(page)
			for i := range resources {
				ch <- Port{
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