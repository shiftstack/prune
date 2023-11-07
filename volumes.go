package main

import (
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/attachments"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/pagination"
)

type Volume struct {
	resource *volumes.Volume
	client   *gophercloud.ServiceClient
}

func (s Volume) CreatedAt() time.Time {
	return s.resource.CreatedAt
}

func (s Volume) Delete() error {
	if s.resource.Attachments != nil {
		s.client.Microversion = "3.44"
		for _, attachment := range s.resource.Attachments {
			err := attachments.Delete(s.client, attachment.AttachmentID).ExtractErr()
			if err != nil {
				return err
			}
		}
	}
	return volumes.Delete(s.client, s.resource.ID, volumes.DeleteOpts{Cascade: true}).ExtractErr()
}

func (s Volume) Type() string {
	return "volume"
}

func (s Volume) ID() string {
	return s.resource.ID
}

func (s Volume) Name() string {
	return s.resource.Name
}

func (s Volume) ClusterID() string {
	return s.resource.Metadata["cinder.csi.openstack.org/cluster"]
}

func ListVolumes(client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := volumes.List(client, nil).EachPage(func(page pagination.Page) (bool, error) {
			resources, err := volumes.ExtractVolumes(page)
			for i := range resources {
				ch <- &Volume{
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
