package main

import (
	"log"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/pagination"
)

type KeyPair struct {
	resource *KeyPairParser
	client   *gophercloud.ServiceClient
}

func (s KeyPair) LastUpdated() time.Time {
	return s.resource.UpdatedAt
}

func (s KeyPair) Delete() error {
	return keypairs.Delete(s.client, s.ID(), keypairs.DeleteOpts{}).ExtractErr()
}

func (s KeyPair) Type() string {
	return "key"
}

func (s KeyPair) ID() string {
	return s.resource.Name
}

func (s KeyPair) Name() string {
	return s.resource.Name
}

type KeyPairParser struct {
	keypairs.KeyPair
	UpdatedAt time.Time `json:"created_at"`
}

func ListKeyPairs(client *gophercloud.ServiceClient) <-chan Resource {
	ch := make(chan Resource)
	go func() {
		defer close(ch)
		if err := keypairs.List(client, nil).EachPage(func(page pagination.Page) (bool, error) {
			keypairPage, err := keypairs.ExtractKeyPairs(page)
			if err != nil {
				return true, err
			}
			for _, keypair := range keypairPage {
				var k struct {
					KeyPairParser `json:"keypair"`
				}
				if err := keypairs.Get(client, keypair.Name, nil).ExtractInto(&k); err != nil {
					return true, err
				}
				log.Println(k)
				ch <- KeyPair{
					resource: &k.KeyPairParser,
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
