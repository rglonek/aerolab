# Exit codes

`aerolab` returns `0` on success and a non-zero code on failure. Most failures
use the generic code `1`; the more specific codes below exist so that scripts
can tell a retryable infrastructure problem apart from a real failure without
having to parse log output.

| Code | Meaning |
|------|---------|
| `0` | The command succeeded. |
| `1` | Generic failure. |
| `10` | AWS or GCP could not supply the requested capacity. |
| `130` | Interrupted (SIGINT). Follows the usual `128 + signal` convention. |
| `143` | Terminated (SIGTERM). |

## Capacity failures (`10`)

Code `10` is returned when instance creation fails because the cloud provider
has no capacity for the requested shape, rather than because the request was
wrong. This covers, among others, `InsufficientInstanceCapacity`,
`VcpuLimitExceeded` and `MaxSpotInstanceCountExceeded` on AWS, and
`ZONE_RESOURCE_POOL_EXHAUSTED`, `QUOTA_EXCEEDED` and `STOCKOUT` on GCP.

These are worth retrying later, in a different zone, or with a different
instance type. `aerolab cluster create` and `aerolab client create` can retry
in-process instead: see `--capacity-retries` and `--capacity-retry-sleep`.

```bash
aerolab cluster create -n asd -c 3
case $? in
  0)  echo "cluster is up" ;;
  10) echo "no capacity right now, try another zone" ;;
  *)  echo "cluster create failed" ;;
esac
```
