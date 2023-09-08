package main

import (
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/pagination"
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

func (s Container) Delete() error {
	_, err := containers.Delete(s.client, s.ID()).Extract()
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

func ListContainers(client *gophercloud.ServiceClient, networks <-chan Resource) <-chan Resource {
	ch := make(chan Resource)
	clusterNetworks := make(map[string]Resource)
	for network := range networks {
		if clusterNetwork, ok := network.(Clusterer); ok && clusterNetwork.ClusterID() != "" {
			clusterNetworks[clusterNetwork.ClusterID()] = network
		}
	}
	go func() {
		defer close(ch)
		if err := containers.List(client, nil).EachPage(func(page pagination.Page) (bool, error) {
			containerPage, err := containers.ExtractNames(page)
			if err != nil {
				return true, err
			}

			for _, containerName := range containerPage {
				getResult := containers.Get(client, containerName, nil)
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
			panic(err)
		}
	}()
	return ch
}
