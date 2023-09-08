package main

import (
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/snapshots"
	"github.com/gophercloud/gophercloud/pagination"
)

type Snapshot struct {
	resource *snapshots.Snapshot
	client   *gophercloud.ServiceClient
}

func (s Snapshot) CreatedAt() time.Time {
	return s.resource.CreatedAt
}

func (s Snapshot) Delete() error {
	return snapshots.Delete(s.client, s.resource.ID).ExtractErr()
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

func ListVolumeSnapshots(client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := snapshots.List(client, nil).EachPage(func(page pagination.Page) (bool, error) {
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
