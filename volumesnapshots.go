package main

import (
	"context"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type Snapshot struct {
	resource *snapshots.Snapshot
	client   *gophercloud.ServiceClient
}

func (s Snapshot) CreatedAt() time.Time {
	return s.resource.CreatedAt
}

func (s Snapshot) Delete(ctx context.Context) error {
	return snapshots.Delete(ctx, s.client, s.resource.ID).ExtractErr()
}

func (s Snapshot) Type() string {
	return "volume snapshot"
}

func (s Snapshot) ID() string {
	return s.resource.ID
}

func (s Snapshot) Name() string {
	return s.resource.Name
}

func ListVolumeSnapshots(ctx context.Context, client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := snapshots.List(client, nil).EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
			resources, err := snapshots.ExtractSnapshots(page)
			for i := range resources {
				ch <- &Snapshot{
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
