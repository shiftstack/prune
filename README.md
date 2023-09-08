# Prune stale OpenShift resources in OpenStack

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
