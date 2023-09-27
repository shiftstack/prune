package main

import (
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/pagination"
)

type LoadBalancer struct {
	resource *loadbalancers.LoadBalancer
	client   *gophercloud.ServiceClient
}

func (s LoadBalancer) CreatedAt() time.Time {
	return s.resource.CreatedAt
}

func (s LoadBalancer) Delete() error {
	return loadbalancers.Delete(s.client, s.resource.ID, loadbalancers.DeleteOpts{Cascade: true}).ExtractErr()
}

func (s LoadBalancer) Type() string {
	return "load balancer"
}

func (s LoadBalancer) ID() string {
	return s.resource.ID
}

func (s LoadBalancer) Name() string {
	return s.resource.Name
}

func (s LoadBalancer) Tags() []string {
	return s.resource.Tags
}

func ListLoadBalancers(client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := loadbalancers.List(client, nil).EachPage(func(page pagination.Page) (bool, error) {
			resources, err := loadbalancers.ExtractLoadBalancers(page)
			for i := range resources {
				ch <- LoadBalancer{
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
