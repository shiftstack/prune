package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type Container struct {
	resourceName       string
	client             *gophercloud.ServiceClient
	clusterID          string
	associatedResource Resource
}

func (s Container) CreatedAt() time.Time {
	if s.associatedResource != nil {
		return s.associatedResource.CreatedAt()
	}
	return time.Time{}
}

func (s Container) Delete(ctx context.Context) error {
	err := objects.List(s.client, s.ID(), &objects.ListOpts{Limit: 50}).
		EachPage(ctx, func(ctx context.Context, page pagination.Page) (bool, error) {
			objectsOnPage, err := objects.ExtractNames(page)
			if err != nil {
				return false, err
			}
			resp, err := objects.BulkDelete(ctx, s.client, s.ID(), objectsOnPage).Extract()
			if err != nil {
				return false, err
			}
			if len(resp.Errors) > 0 {
				// Convert resp.Errors to golang errors.
				// Each error is represented by a list of 2 strings, where the first one
				// is the object name, and the second one contains an error message.
				errs := make([]error, len(resp.Errors))
				for i, objectError := range resp.Errors {
					errs[i] = fmt.Errorf("cannot delete object %s: %s", objectError[0], objectError[1])
				}

				return false, fmt.Errorf("errors occurred during bulk deleting of container %s objects: %v", s.ID(), errs)
			}

			return true, nil
		})
	if err != nil {
		var gerr gophercloud.ErrDefault404
		if !errors.As(err, &gerr) {
			log.Printf("Bulk deleting of container %q objects failed: %v", s.ID(), err)
			return err
		}
	}
	log.Printf("Deleting container %q", s.ID())
	_, err = containers.Delete(ctx, s.client, s.ID()).Extract()
	if err != nil {
		// Ignore the error if the container cannot be found and return with an appropriate message if it's another type of error
		var gerr gophercloud.ErrDefault404
		if !errors.As(err, &gerr) {
			log.Printf("Deleting container %q failed: %v", s.ID(), err)
			return err
		}
		log.Printf("Cannot find container %q. It's probably already been deleted.", s.ID())
	}

	return err
}

func (s Container) Type() string {
	return "container"
}

func (s Container) ID() string {
	return s.resourceName
}

func (s Container) Name() string {
	return s.resourceName
}

func (s Container) ClusterID() string {
	return s.clusterID
}

type ContainerParser struct {
	containers.Container
	Properties map[string]string `json:"properties"`
}

func ListContainers(ctx context.Context, client *gophercloud.ServiceClient, networks <-chan Resource) <-chan Resource {
	ch := make(chan Resource)
	clusterNetworks := make(map[string]Resource)
	for network := range networks {
		if clusterNetwork, ok := network.(Clusterer); ok && clusterNetwork.ClusterID() != "" {
			clusterNetworks[clusterNetwork.ClusterID()] = network
		}
	}
	go func() {
		defer close(ch)
		if err := containers.List(client, nil).EachPage(ctx, func(ctx context.Context, page pagination.Page) (bool, error) {
			containerPage, err := containers.ExtractNames(page)
			if err != nil {
				return true, err
			}

			for _, containerName := range containerPage {
				getResult := containers.Get(ctx, client, containerName, nil)
				if getResult.Err != nil {
					return true, err
				}
				c := Container{
					resourceName: containerName,
					client:       client,
					clusterID:    getResult.Header.Get("X-Container-Meta-Openshiftclusterid"),
				}
				if n, ok := clusterNetworks[c.ClusterID()]; ok {
					c.associatedResource = n
				}
				ch <- c
			}
			return true, nil
		}); err != nil {
			var errForbidden gophercloud.ErrDefault403
			if errors.As(err, &errForbidden) {
				log.Printf("Skipping containers deletion. User not authorized to perform the requested action")
			} else {
				panic(err)
			}
		}
	}()
	return ch
}
