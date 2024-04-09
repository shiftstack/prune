package main

import (
	"context"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type SecurityGroup struct {
	resource *groups.SecGroup
	client   *gophercloud.ServiceClient
}

func (s SecurityGroup) CreatedAt() time.Time {
	return s.resource.CreatedAt
}

func (s SecurityGroup) Delete(ctx context.Context) error {
	return groups.Delete(ctx, s.client, s.resource.ID).ExtractErr()
}

func (s SecurityGroup) Type() string {
	return "security group"
}

func (s SecurityGroup) ID() string {
	return s.resource.ID
}

func (s SecurityGroup) Name() string {
	return s.resource.Name
}

func (s SecurityGroup) Tags() []string {
	return s.resource.Tags
}

func (s SecurityGroup) ClusterID() string {
	for _, tag := range s.resource.Tags {
		if value := strings.TrimPrefix(tag, "openshiftClusterID="); value != tag {
			return value
		}
	}
	return ""
}

func ListSecurityGroups(ctx context.Context, client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := groups.List(client, groups.ListOpts{
			TenantID: "",
		}).EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
			resources, err := groups.ExtractGroups(page)
			for i := range resources {
				ch <- SecurityGroup{
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
