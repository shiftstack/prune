package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/term"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/utils/openstack/clientconfig"
)

const helpString = `Prune stale resources from cloud

Usage: prune [OPTION]...

Options:
  --resource-ttl=<ttl>   Minimum age of resources to prune. ttl is parsed
                         as a Go duration (e.g. "1h", "30m12s")
  --ignore=<ignore-path> A JSON file containing a list of resources to ignore
                         when pruning. Each entry should contain a 'type' key
                         and either a 'name' or 'id' key.
  --slack-hook=<hook>    Slack hook. If provided, updates will be posted to the
                         relevant Slack channel, otherwise they are dumped to
                         stdout
  --no-dry-run           Delete resources
  --help                 Show this help message and exit

Examples:

  Given an ignore.json file like so:

    [
      {
        "id": "36ac32c1-062e-44c4-b833-2421602c095d",
        "name": "199.94.60.197",
        "type": "floating ip"
      },
      {
        "id": "d2929e15-8452-4969-972a-a619d9a87854",
        "name": "199.94.60.52",
        "type": "floating ip"
      }
    ]

  If you wanted to delete all resources older than 1 hour 30 mins, you can
  first run the following to see what will be deleted:

    prune --ignore=ignore.json --resource-ttl=1h30s

  Once you are satisfied with the resource deletion, you can pass the
  --no-dry-run flag.

    prune --ignore=ignore.json --resource-ttl=1h30s --no-dry-run
`

var showHelp = func() bool {
	for _, arg := range os.Args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}()

type IgnoredResource struct {
	ID   *string `json:"id,omitempty"`
	Name *string `json:"name,omitempty"`
	Type *string `json:"type"`
}

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

var slackHook = func() string {
	for _, arg := range os.Args {
		if value := strings.TrimPrefix(arg, "--slack-hook="); value != arg {
			return value
		}
	}
	return ""
}()

var ignoredResources = func() []IgnoredResource {
	for _, arg := range os.Args {
		if value := strings.TrimPrefix(arg, "--ignore="); value != arg {
			if _, err := os.Stat(value); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Error: Ignore file does not exist: %s\n", value)
				os.Exit(1)
			}

			jsonFile, err := os.Open(value)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to read ignore file: %+v\n", err)
			}
			defer jsonFile.Close()

			byteValue, _ := io.ReadAll(jsonFile)
			var ignoredResources []IgnoredResource
			json.Unmarshal(byteValue, &ignoredResources)
			return ignoredResources
		}
	}
	return []IgnoredResource{}
}()

type Resource interface {
	Dater
	Deleter
	Identifier
	Namer
	Typer
}

type Typer interface{ Type() string }
type Dater interface{ CreatedAt() time.Time }
type Deleter interface{ Delete() error }
type Identifier interface{ ID() string }
type Namer interface{ Name() string }
type Clusterer interface{ ClusterID() string }
type Tagger interface{ Tags() []string }

// TODO: server groups, keypairs
// TODO: volume admin setting
func main() {
	if showHelp {
		fmt.Printf(helpString)
		os.Exit(0)
	}

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
			// Ignore the error if Octavia is not available in the cloud
			var gerr *gophercloud.ErrEndpointNotFound
			if errors.As(err, &gerr) {
				log.Println("Skipping load balancer listing because the Octavia endpoint was not found")
			} else {
				panic(err)
			}
			loadbalancerClient = nil
		}
		computeClient, err := clientconfig.NewServiceClient("compute", &opts)
		if err != nil {
			panic(err)
		} else {
			// Required for server tags
			computeClient.Microversion = "2.26"
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
		imageClient, err := clientconfig.NewServiceClient("image", &opts)
		if err != nil {
			panic(err)
		}

		containerClient, err := clientconfig.NewServiceClient("object-store", &opts)
		if err != nil {
			// Ignore the error if Swift is not available in the cloud
			var gerr *gophercloud.ErrEndpointNotFound
			if errors.As(err, &gerr) {
				log.Println("Skipping container listing because the Swift endpoint was not found")
			} else {
				panic(err)
			}
			containerClient = nil
		}

		shareClient, err := clientconfig.NewServiceClient("sharev2", &opts)
		if err != nil {
			// Ignore the error if Manila is not available in the cloud
			var gerr *gophercloud.ErrEndpointNotFound
			if errors.As(err, &gerr) {
				log.Println("Skipping share listing because the Manila endpoint was not found")
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

			if loadbalancerClient != nil {
				for res := range ListLoadBalancers(loadbalancerClient) {
					resources <- res
				}
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

			for res := range Filter(ListNetworks(networkClient), NameDoesNotContain[Resource]("hostonly", "external", "sahara-access", "mellanox", "intel", "public", "provider")) {
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

			if containerClient != nil {
				for res := range Filter(ListContainers(containerClient, ListNetworks(networkClient)), NameIsNot[Resource]("shiftstack-metrics", "shiftstack-bot")) {
					resources <- res
				}
			}

			for res := range Filter(ListImages(imageClient), NameMatchesOneOfThesePatterns[Resource](".{8}-.{5}-.{5}-ignition", ".{8}-.{5}-.{5}-rhcos", "bootstrap-ign-.{8}-.{5}-.{5}", "rhcos-.{7,8}-.{5}")) {
				resources <- res
			}
		}()
	}

	now := time.Now()
	report := Report{Time: now}
	for staleResource := range Filter(resources, TagsDoNotContain("shiftstack-prune=keep"), CreatedBefore[Resource](now.Add(-bestBefore)), IsNotIgnored(ignoredResources)) {
		report.AddFound(staleResource)

		if !dryRun {
			log.Printf("Deleting %s %q (created at %s)...\n", staleResource.Type(), staleResource.ID(), staleResource.CreatedAt().Format(time.RFC3339))
			if err := staleResource.Delete(); err != nil {
				log.Printf("error deleting %s %q: %v\n", staleResource.Type(), staleResource.ID(), err)
				report.AddFailedToDelete(staleResource)
			} else {
				log.Printf("deleted %s %q\n", staleResource.Type(), staleResource.ID())
				report.AddDeleted(staleResource)
			}
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		encoder.SetIndent("", "  ")
	}

	if err := encoder.Encode(report); err != nil {
		panic(err)
	}

	if len(report.FailedToDelete) > 0 && slackHook != "" {
		log.Printf("Sending failed_to_delete report to Slack")
		if err := reportToSlack(slackHook, report); err != nil {
			log.Fatalf("Failed to send a report to Slack: %v", err)
		}
	}
}

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)
}
