package main

import (
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/pagination"
)

type Trunk struct {
	resource *trunks.Trunk
	client   *gophercloud.ServiceClient
}

func (s Trunk) CreatedAt() time.Time {
	return s.resource.CreatedAt
}

func (s Trunk) Delete() error {
	return trunks.Delete(s.client, s.resource.ID).ExtractErr()
}

func (s Trunk) Type() string {
	return "trunk"
}

func (s Trunk) ID() string {
	return s.resource.ID
}

func (s Trunk) Name() string {
	return s.resource.Name
}

func (s Trunk) ClusterID() string {
	for _, tag := range s.resource.Tags {
		if value := strings.TrimPrefix(tag, "openshiftClusterID="); value != tag {
			return value
		}
	}
	return ""
}

func ListTrunks(client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := trunks.List(client, nil).EachPage(func(page pagination.Page) (bool, error) {
			resources, err := trunks.ExtractTrunks(page)
			for i := range resources {
				ch <- Trunk{
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
