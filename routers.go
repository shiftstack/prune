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

func (s Router) LastUpdated() time.Time {
	return s.resource.UpdatedAt
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
	UpdatedAt time.Time `json:"updated_at"`
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

			for _, router := range routerPage.Routers {
				if err := ports.List(client, ports.ListOpts{DeviceID: router.ID}).EachPage(func(page pagination.Page) (bool, error) {
					portList, err := ports.ExtractPorts(page)
					if err != nil {
						return false, err
					}
					for _, port := range portList {
						for _, fixedIP := range port.FixedIPs {
							router.subnets = append(router.subnets, fixedIP.SubnetID)
						}
					}
					return true, nil
				}); err != nil {
					return true, err
				}
				ch <- Router{
					resource: &router,
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
