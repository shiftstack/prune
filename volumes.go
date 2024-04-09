package main

import (
	"context"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/attachments"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type Volume struct {
	resource *volumes.Volume
	client   *gophercloud.ServiceClient
}

func (s Volume) CreatedAt() time.Time {
	return s.resource.CreatedAt
}

func (s Volume) Delete(ctx context.Context) error {
	if s.resource.Attachments != nil {
		s.client.Microversion = "3.44"
		for _, attachment := range s.resource.Attachments {
			err := attachments.Delete(ctx, s.client, attachment.AttachmentID).ExtractErr()
			if err != nil {
				return err
			}
		}
	}
	return volumes.Delete(ctx, s.client, s.resource.ID, volumes.DeleteOpts{Cascade: true}).ExtractErr()
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

func ListVolumes(ctx context.Context, client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := volumes.List(client, nil).EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
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
