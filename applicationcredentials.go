package main

import (
	"context"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/applicationcredentials"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type ApplicationCredential struct {
	resource *applicationcredentials.ApplicationCredential
	client   *gophercloud.ServiceClient
	userID   string
}

func (s ApplicationCredential) CreatedAt() time.Time {
	return s.resource.ExpiresAt
}

func (s ApplicationCredential) Delete(ctx context.Context) error {
	return applicationcredentials.Delete(ctx, s.client, s.userID, s.resource.ID).ExtractErr()
}

func (s ApplicationCredential) Type() string {
	return "application credential"
}

func (s ApplicationCredential) ID() string {
	return s.resource.ID
}

func (s ApplicationCredential) Name() string {
	return s.resource.Name
}

func (s ApplicationCredential) ClusterID() string {
	for _, tag := range strings.Split(s.resource.Description, " ") {
		// https://github.com/openshift/release/pull/43348
		if value := strings.TrimPrefix(tag, "PROW_CLUSTER_NAME="); value != tag {
			return value
		}
	}
	return ""
}

func getUserID(ctx context.Context, client *gophercloud.ServiceClient) (string, error) {
	var token struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	err := tokens.Get(ctx, client, client.Token()).ExtractInto(&token)
	return token.User.ID, err
}

func ListPerishableApplicationCredentials(ctx context.Context, client *gophercloud.ServiceClient) <-chan Resource {
	userID, err := getUserID(ctx, client)
	if err != nil {
		panic(err)
	}
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := applicationcredentials.List(client, userID, nil).EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
			resources, err := applicationcredentials.ExtractApplicationCredentials(page)
			for i := range resources {
				if !resources[i].ExpiresAt.IsZero() {
					ch <- ApplicationCredential{
						resource: &resources[i],
						client:   client,
						userID:   userID,
					}
				}
			}
			return true, err
		}); err != nil {
			panic(err)
		}
	}()
	return ch
}
