package main

// import (
// 	"github.com/gophercloud/gophercloud"
// 	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
// 	"github.com/gophercloud/gophercloud/pagination"
// )

// type Container struct {
// 	resource string
// 	client   *gophercloud.ServiceClient
// }

// func (s Container) Delete() error {
// 	_, err := containers.Delete(s.client, s.ID()).Extract()
// 	return err
// }

// func (s Container) Type() string {
// 	return "container"
// }

// func (s Container) ID() string {
// 	return s.resource
// }

// func ListContainers(client *gophercloud.ServiceClient) <-chan Resource {
// 	ch := make(chan Resource)
// 	go func() {
// 		defer close(ch)
// 		if err := containers.List(client, nil).EachPage(func(page pagination.Page) (bool, error) {
// 			resources, err := containers.ExtractNames(page)

// 			for _, name := range resources {
// 				ch <- Container{
// 					resource: name,
// 					client:   client,
// 				}
// 			}
// 			return true, err
// 		}); err != nil {
// 			panic(err)
// 		}
// 	}()
// 	return ch
// }
