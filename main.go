package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
)

const ConsideredStale = 5 * time.Second

var dryRun = true

type Resource interface {
	Updater
	Deleter
	Identifier
	Namer
	Typer
}

type Typer interface{ Type() string }
type Updater interface{ LastUpdated() time.Time }
type Deleter interface{ Delete() error }
type Identifier interface{ ID() string }
type Namer interface{ Name() string }
type Clusterer interface{ ClusterID() string }

// func runClusterDestroy(provider *gophercloud.ProviderClient, endpointOpts gophercloud.EndpointOpts) {
// 	networkClient, err := openstack.NewNetworkV2(provider, endpointOpts)
// 	if err != nil {
// 		panic(err)
// 	}
// }

// TODO: containers, keypairs, images
// TODO: volume admin setting
func main() {
	resources := make(chan Resource)
	{
		ao, err := clientconfig.AuthOptions(&clientconfig.ClientOpts{
			Cloud: os.Getenv("OS_CLOUD"),
		})
		if err != nil {
			panic(err)
		}
		provider, err := openstack.AuthenticatedClient(*ao)
		if err != nil {
			panic(err)
		}
		endpointOpts := gophercloud.EndpointOpts{
			Region: os.Getenv("OS_REGION_NAME"),
		}

		loadbalancerClient, err := openstack.NewLoadBalancerV2(provider, endpointOpts)
		if err != nil {
			panic(err)
		}
		networkClient, err := openstack.NewNetworkV2(provider, endpointOpts)
		if err != nil {
			panic(err)
		}
		computeClient, err := openstack.NewComputeV2(provider, endpointOpts)
		if err != nil {
			panic(err)
		}
		volumeClient, err := openstack.NewBlockStorageV3(provider, endpointOpts)
		if err != nil {
			panic(err)
		}

		go func() {
			defer close(resources)

			for res := range ListLoadBalancers(loadbalancerClient) {
				resources <- res
			}

			for res := range Filter(ListServers(computeClient), NameIsNot[Resource]("metrics")) {
				resources <- res
			}

			for res := range Filter(ListRouters(networkClient), NameIsNot[Resource]("dualstack")) {
				resources <- res
			}

			for res := range Filter(ListNetworks(networkClient), NameDoesNotContain[Resource]("hostonly", "external", "sahara-access", "mellanox", "intel", "public", "slaac")) {
				resources <- res
			}

			for res := range ListVolumeSnapshots(volumeClient) {
				resources <- res
			}

			for res := range ListVolumes(volumeClient) {
				resources <- res
			}

			for res := range ListFloatingIPs(networkClient) {
				resources <- res
			}

			for res := range Filter(ListSecurityGroups(networkClient), NameIsNot[Resource]("default", "ssh", "allow_ssh", "allow_ping")) {
				resources <- res
			}
		}()
	}

	now := time.Now()
	report := Report{CreatedAt: now}
	for staleResource := range Filter(resources, InactiveSince[Resource](now.Add(-ConsideredStale))) {
		report.AddFound(staleResource)

		if !dryRun {
			log.Printf("Deleting %s %q (updated %s)...\n", staleResource.Type(), staleResource.ID(), staleResource.LastUpdated().Format(time.RFC3339))
			if err := staleResource.Delete(); err != nil {
				log.Printf("error deleting %s %q: %v\n", staleResource.Type(), staleResource.ID(), err)
				report.AddFailedToDelete(staleResource)
			} else {
				log.Printf("deleted %s %q\n", staleResource.Type(), staleResource.ID())
				report.AddDeleted(staleResource)
			}
		}
	}

	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		panic(err)
	}
}

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)
}
