package main

import (
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/applicationcredentials"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/pagination"
)

type ApplicationCredential struct {
	resource *applicationcredentials.ApplicationCredential
	client   *gophercloud.ServiceClient
	userID   string
}

func (s ApplicationCredential) CreatedAt() time.Time {
	return s.resource.ExpiresAt
}

func (s ApplicationCredential) Delete() error {
	return applicationcredentials.Delete(s.client, s.userID, s.resource.ID).ExtractErr()
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

func getUserID(client *gophercloud.ServiceClient) (string, error) {
	var token struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	err := tokens.Get(client, client.Token()).ExtractInto(&token)
	return token.User.ID, err
}

func ListPerishableApplicationCredentials(client *gophercloud.ServiceClient) <-chan Resource {
	userID, err := getUserID(client)
	if err != nil {
		panic(err)
	}
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := applicationcredentials.List(client, userID, nil).EachPage(func(page pagination.Page) (bool, error) {
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
