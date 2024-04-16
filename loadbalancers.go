package main

import (
	"context"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type LoadBalancer struct {
	resource *loadbalancers.LoadBalancer
	client   *gophercloud.ServiceClient
}

func (s LoadBalancer) CreatedAt() time.Time {
	return s.resource.CreatedAt
}

func (s LoadBalancer) Delete(ctx context.Context) error {
	return loadbalancers.Delete(ctx, s.client, s.resource.ID, loadbalancers.DeleteOpts{Cascade: true}).ExtractErr()
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

func ListLoadBalancers(ctx context.Context, client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := loadbalancers.List(client, nil).EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
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
