package main

import (
	"context"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/shares"
	sharesnapshots "github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/snapshots"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type Share struct {
	resource *shares.Share
	client   *gophercloud.ServiceClient
}

func (s Share) CreatedAt() time.Time {
	return s.resource.CreatedAt
}

func (s Share) Delete(ctx context.Context) error {
	if err := deleteShareSnapshots(ctx, s.client, s.ID()); err != nil {
		return err
	}
	return shares.Delete(ctx, s.client, s.resource.ID).ExtractErr()
}

func (s Share) Type() string {
	return "share"
}

func (s Share) ID() string {
	return s.resource.ID
}

func (s Share) Name() string {
	return s.resource.Name
}

func (s Share) ClusterID() string {
	return s.resource.Metadata["manila.csi.openstack.org/cluster"]
}

func ListShares(ctx context.Context, client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := shares.ListDetail(client, nil).EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
			resources, err := shares.ExtractShares(page)
			for i := range resources {
				ch <- Share{
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

func deleteShareSnapshots(ctx context.Context, conn *gophercloud.ServiceClient, shareID string) error {
	listOpts := sharesnapshots.ListOpts{
		ShareID: shareID,
	}

	return sharesnapshots.ListDetail(conn, listOpts).EachPage(ctx, func(ctx context.Context, page pagination.Page) (bool, error) {
		allSnapshots, err := sharesnapshots.ExtractSnapshots(page)
		if err != nil {
			return true, err
		}

		for _, snapshot := range allSnapshots {
			if err := sharesnapshots.Delete(ctx, conn, snapshot.ID).ExtractErr(); err != nil {
				return true, err
			}
		}
		return true, nil
	})
}
