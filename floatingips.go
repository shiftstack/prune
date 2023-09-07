package main

import (
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/pagination"
)

type FloatingIP struct {
	resource *floatingips.FloatingIP
	client   *gophercloud.ServiceClient
}

func (s FloatingIP) LastUpdated() time.Time {
	return s.resource.UpdatedAt
}

func (s FloatingIP) Delete() error {
	return floatingips.Delete(s.client, s.resource.ID).ExtractErr()
}

func (s FloatingIP) Type() string {
	return "floating ip"
}

func (s FloatingIP) ID() string {
	return s.resource.ID
}

func (s FloatingIP) Name() string {
	return s.resource.FloatingIP
}

func (s FloatingIP) ClusterID() string {
	for _, tag := range s.resource.Tags {
		if value := strings.TrimPrefix(tag, "openshiftClusterID="); value != tag {
			return value
		}
		// https://github.com/openshift/release/pull/43063
		if value := strings.TrimPrefix(tag, "PROW_CLUSTER_NAME="); value != tag {
			return value
		}
	}
	return ""
}

func ListFloatingIPs(client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := floatingips.List(client, floatingips.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
			floatingIPPage, err := floatingips.ExtractFloatingIPs(page)

			for i := range floatingIPPage {
				ch <- FloatingIP{
					resource: &floatingIPPage[i],
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
