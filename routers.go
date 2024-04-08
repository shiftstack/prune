package main

import (
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/pagination"
)

type Router struct {
	resource *RouterParser
	client   *gophercloud.ServiceClient
}

func (s Router) CreatedAt() time.Time {
	return s.resource.CreatedAt
}

func (s Router) Delete() error {
	for _, subnet := range s.resource.subnets {
		if _, err := routers.RemoveInterface(s.client, s.resource.ID, routers.RemoveInterfaceOpts{SubnetID: subnet}).Extract(); err != nil {
			return err
		}
	}
	return routers.Delete(s.client, s.resource.ID).ExtractErr()
}

func (s Router) Type() string {
	return "router"
}

func (s Router) ID() string {
	return s.resource.ID
}

func (s Router) Name() string {
	return s.resource.Name
}

func (s Router) Tags() []string {
	return s.resource.Tags
}

func (s Router) ClusterID() string {
	for _, tag := range s.resource.Tags {
		if value := strings.TrimPrefix(tag, "openshiftClusterID="); value != tag {
			return value
		}
	}
	return ""
}

type RouterParser struct {
	routers.Router
	CreatedAt time.Time `json:"created_at"`
	subnets   []string
}

func ListRouters(client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := routers.List(client, routers.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
			var routerPage struct {
				Routers []RouterParser `json:"routers"`
			}

			err := (page.(routers.RouterPage)).ExtractInto(&routerPage)
			if err != nil {
				return true, err
			}

			for i := range routerPage.Routers {
				if err := ports.List(client, ports.ListOpts{DeviceID: routerPage.Routers[i].ID, DeviceOwner: "network:router_interface"}).EachPage(func(page pagination.Page) (bool, error) {
					portList, err := ports.ExtractPorts(page)
					if err != nil {
						return false, err
					}
					for _, port := range portList {
						for j := range port.FixedIPs {
							routerPage.Routers[i].subnets = append(routerPage.Routers[i].subnets, port.FixedIPs[j].SubnetID)
						}
					}
					return true, nil
				}); err != nil {
					return true, err
				}
				ch <- Router{
					resource: &routerPage.Routers[i],
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
