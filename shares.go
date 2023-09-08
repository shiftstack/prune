package main

import (
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/sharedfilesystems/v2/shares"
	sharesnapshots "github.com/gophercloud/gophercloud/openstack/sharedfilesystems/v2/snapshots"
	"github.com/gophercloud/gophercloud/pagination"
)

type Share struct {
	resource *shares.Share
	client   *gophercloud.ServiceClient
}

func (s Share) LastUpdated() time.Time {
	return s.resource.UpdatedAt
}

func (s Share) Delete() error {
	if err := deleteShareSnapshots(s.client, s.ID()); err != nil {
		return err
	}
	return shares.Delete(s.client, s.resource.ID).ExtractErr()
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

func ListShares(client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := shares.ListDetail(client, nil).EachPage(func(page pagination.Page) (bool, error) {
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

func deleteShareSnapshots(conn *gophercloud.ServiceClient, shareID string) error {
	listOpts := sharesnapshots.ListOpts{
		ShareID: shareID,
	}

	return sharesnapshots.ListDetail(conn, listOpts).EachPage(func(page pagination.Page) (bool, error) {
		allSnapshots, err := sharesnapshots.ExtractSnapshots(page)
		if err != nil {
			return true, err
		}

		for _, snapshot := range allSnapshots {
			if err := sharesnapshots.Delete(conn, snapshot.ID).ExtractErr(); err != nil {
				return true, err
			}
		}
		return true, nil
	})
}
