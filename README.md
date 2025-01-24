# Prune stale OpenShift resources in OpenStack

List resources older than a threshold.

Ignores resources tagged with: `shiftstack-prune=keep`.

## Use

Dry run:
```shell
export OS_CLOUD=<clouds.yaml entry>
./prune
```

Actual run:
```shell
export OS_CLOUD=<clouds.yaml entry>
./prune --no-dry-run
```

Configure the resoruce TTL with `--resource-ttl=<duration>` where `<duration>` is expressed as a Go duration. For example:
```shell
export OS_CLOUD=<clouds.yaml entry>
./prune --no-dry-run --resource-ttl=5h
```

## Resource filtering

Filter resources by type:

```shell
# Only process servers and volumes
./prune --include=servers,volumes

# Process everything except images and networks
./prune --exclude=images,networks

Available resource types:

| Resource type    | Service    | Description                        |
|------------------|------------|------------------------------------|
| `appcreds`       | `keystone` | Application credentials            |
| `containers`     | `swift`    | Object storage containers          |
| `floatingips`    | `neutron`  | Public IP addresses               |
| `images`         | `glance`   | Virtual machine images            |
| `loadbalancers`  | `octavia`  | Load balancers                    |
| `networks`       | `neutron`  | Virtual networks                  |
| `ports`          | `neutron`  | Virtual network ports             |
| `routers`        | `neutron`  | Virtual routers                   |
| `securitygroups` | `neutron`  | Security groups                   |
| `servers`        | `nova`     | Virtual machines                  |
| `shares`         | `manila`   | Shared file systems               |
| `trunks`         | `neutron`  | Virtual network trunks            |
| `volumes`        | `cinder`   | Block storage volumes             |
| `volumesnapshots`| `cinder`   | Block storage volume snapshots    |
