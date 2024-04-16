package main

import (
	"context"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/trunks"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type Trunk struct {
	resource *trunks.Trunk
	client   *gophercloud.ServiceClient
}

func (s Trunk) CreatedAt() time.Time {
	return s.resource.CreatedAt
}

func (s Trunk) Delete(ctx context.Context) error {
	return trunks.Delete(ctx, s.client, s.resource.ID).ExtractErr()
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

func (s Trunk) Tags() []string {
	return s.resource.Tags
}

func (s Trunk) ClusterID() string {
	for _, tag := range s.resource.Tags {
		if value := strings.TrimPrefix(tag, "openshiftClusterID="); value != tag {
			return value
		}
	}
	return ""
}

func ListTrunks(ctx context.Context, client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := trunks.List(client, nil).EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
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
