package main

import (
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/gophercloud/pagination"
)

type Image struct {
	resource *images.Image
	client   *gophercloud.ServiceClient
}

func (s Image) CreatedAt() time.Time {
	return s.resource.CreatedAt
}

func (s Image) Delete() error {
	return images.Delete(s.client, s.resource.ID).ExtractErr()
}

func (s Image) Type() string {
	return "image"
}

func (s Image) ID() string {
	return s.resource.ID
}

func (s Image) Name() string {
	return s.resource.Name
}

func (s Image) ClusterID() string {
	for _, tag := range s.resource.Tags {
		// https://github.com/openshift/installer/blob/75ac0821ee012d8855dadf42c25cc807d8ef8d51/pkg/tfvars/openstack/rhcos_image.go#L68
		if value := strings.TrimPrefix(tag, "openshiftClusterID="); value != tag {
			return value
		}
	}
	return ""
}

func ListImages(client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := images.List(client, nil).EachPage(func(page pagination.Page) (bool, error) {
			resources, err := images.ExtractImages(page)
			for i := range resources {
				ch <- Image{
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
