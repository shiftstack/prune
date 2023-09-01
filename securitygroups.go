package main

import (
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/pagination"
)

type SecurityGroup struct {
	resource *groups.SecGroup
	client   *gophercloud.ServiceClient
}

func (s SecurityGroup) LastUpdated() time.Time {
	return s.resource.UpdatedAt
}

func (s SecurityGroup) Delete() error {
	return groups.Delete(s.client, s.resource.ID).ExtractErr()
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

func (s SecurityGroup) ClusterID() string {
	for _, tag := range s.resource.Tags {
		if value := strings.TrimPrefix(tag, "openshiftClusterID="); value != tag {
			return value
		}
	}
	return ""
}

func ListSecurityGroups(client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := groups.List(client, groups.ListOpts{
			TenantID: "",
		}).EachPage(func(page pagination.Page) (bool, error) {
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
