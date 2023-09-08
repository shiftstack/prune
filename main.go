package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/utils/openstack/clientconfig"
)

var bestBefore = func() time.Duration {
	for _, arg := range os.Args {
		if value := strings.TrimPrefix(arg, "--resource-ttl="); value != arg {
			d, err := time.ParseDuration(value)
			if err != nil {
				panic(err)
			}
			return d
		}
	}
	return 7 * time.Hour
}()

var dryRun = func() bool {
	for _, arg := range os.Args {
		if arg == "--no-dry-run" {
			return false
		}
	}
	return true
}()

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

// TODO:  keypairs
// TODO: volume admin setting
func main() {
	{
		verb := "Listing"
		if !dryRun {
			verb = "Deleting"
		}
		log.Printf("%s everything older than %s\n", verb, bestBefore)
	}
	resources := make(chan Resource)
	{
		opts := clientconfig.ClientOpts{Cloud: os.Getenv("OS_CLOUD")}
		loadbalancerClient, err := clientconfig.NewServiceClient("load-balancer", &opts)
		if err != nil {
			panic(err)
		}
		computeClient, err := clientconfig.NewServiceClient("compute", &opts)
		if err != nil {
			panic(err)
		}
		networkClient, err := clientconfig.NewServiceClient("network", &opts)
		if err != nil {
			panic(err)
		}
		volumeClient, err := clientconfig.NewServiceClient("volume", &opts)
		if err != nil {
			panic(err)
		}
		identityClient, err := clientconfig.NewServiceClient("identity", &opts)
		if err != nil {
			panic(err)
		}
		containerClient, err := clientconfig.NewServiceClient("object-store", &opts)
		if err != nil {
			panic(err)
		}
		imageClient, err := clientconfig.NewServiceClient("image", &opts)
		if err != nil {
			panic(err)
		}

		shareClient, err := clientconfig.NewServiceClient("sharev2", &opts)
		if err != nil {
			// Ignore the error if Manila is not available in the cloud
			var gerr *gophercloud.ErrEndpointNotFound
			if errors.As(err, &gerr) {
				log.Println("Skipping share deletion because Manila endpoint was not found")
			} else {
				panic(err)
			}
			shareClient = nil
		}

		go func() {
			defer close(resources)

			for res := range ListFloatingIPs(networkClient) {
				resources <- res
			}

			for res := range ListLoadBalancers(loadbalancerClient) {
				resources <- res
			}

			for res := range Filter(ListServers(computeClient), NameIsNot[Resource]("metrics")) {
				resources <- res
			}

			for res := range Filter(ListRouters(networkClient), NameIsNot[Resource]("dualstack")) {
				resources <- res
			}

			for res := range ListTrunks(networkClient) {
				resources <- res
			}

			for res := range ListPorts(networkClient) {
				resources <- res
			}

			for res := range Filter(ListNetworks(networkClient), NameDoesNotContain[Resource]("hostonly", "external", "sahara-access", "mellanox", "intel", "public")) {
				resources <- res
			}

			for res := range ListVolumeSnapshots(volumeClient) {
				resources <- res
			}

			for res := range ListVolumes(volumeClient) {
				resources <- res
			}

			for res := range Filter(ListSecurityGroups(networkClient), NameIsNot[Resource]("default", "ssh", "allow_ssh", "allow_ping")) {
				resources <- res
			}

			if shareClient != nil {
				for res := range ListShares(shareClient) {
					resources <- res
				}
			}

			for res := range ListPerishableApplicationCredentials(identityClient) {
				resources <- res
			}

			for res := range Filter(ListContainers(containerClient, ListNetworks(networkClient)), NameIsNot[Resource]("shiftstack-metrics", "shiftstack-bot")) {
				resources <- res
			}

			for res := range Filter(ListImages(imageClient), NameMatchesOneOfThesePatterns[Resource](".{8}-.{5}-.{5}-ignition", ".{8}-.{5}-.{5}-rhcos", "bootstrap-ign-.{8}-.{5}-.{5}", "rhcos-.{7,8}-.{5}")) {
				resources <- res
			}
		}()
	}

	now := time.Now()
	report := Report{Time: now}
	for staleResource := range Filter(resources, InactiveSince[Resource](now.Add(-bestBefore))) {
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
