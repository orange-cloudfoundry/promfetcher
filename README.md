# Promfetcher

Promfetcher was made for [cloud foundry](https://cloudfoundry.org) and the idea behind is to give ability to fetch 
metrics from all app instances in a cloud foundry environment.

User can retrieve is metrics by simply call `/v1/apps/[org_name]/[space_name]/[app_name]/metrics` which will merge all metrics 
from instances and add labels:
- `organization_id`
- `space_id` 
- `app_id`
- `organization_name`
- `space_name` 
- `app_name`
- `index` - app instance index
- `instance_id` - the same as index
- `instance` - real container address

It also a service broker for cloud foundry to be able to set metrics endpoint for a particular which not use `/metrics` by default.

## Example

Metrics from app instance 0:
```
go_memstats_mspan_sys_bytes{} 65536
```

Metrics from app instance 1:
```
go_memstats_mspan_sys_bytes{} 5600
```

become:
```
go_memstats_mspan_sys_bytes{organization_id="7d66c7e7-196a-40e5-a259-f5afaf6a56f4",space_id="2ac205af-e18f-49a9-9a8b-48ef2bab2292",app_id="621617db-9dd9-4211-8848-b245f3ea16b2",organization_name="system",space_name="tools",app_name="app",index="0",instance_id="0",instance="172.76.112.90:61038"} 65536
go_memstats_mspan_sys_bytes{organization_id="7d66c7e7-196a-40e5-a259-f5afaf6a56f4",space_id="2ac205af-e18f-49a9-9a8b-48ef2bab2292",app_id="621617db-9dd9-4211-8848-b245f3ea16b2",organization_name="system",space_name="tools",app_name="app",index="1",instance_id="1",instance="172.76.112.91:61010"} 65536
```

## How to use ?

See [how to use](/userdocs/templates/how-to-use.md) file which is also readable as a user after deploy promfetcher 
at http://promefetcher.url.com/doc

## How it works ?

Promfetcher only needs [gorouter](https://github.com/cloudfoundry/gorouter) and will read route table from it.

When asking metrics for an app, promfetcher will call async all app instance (gave by routing table from gorouter) metrics endpoint and merge them together with new labels.

## How to deploy ?

You should deploy it with boshrelease associated with: https://github.com/orange-cloudfoundry/promfetcher-release

## Metrics

Promfetcher expose metrics on `/metrics`:
- `promfetch_metric_fetch_failed_total`: Number of non fetched metrics without be an normal error.
- `promfetch_metric_fetch_success_total`: Number of fetched metrics succeeded for an app (app instance call are summed).
- `promfetch_latest_time_scrape_route`: Last time that route has been scraped in seconds.
- `promfetch_scrape_route_failed_total`: Number of non fetched metrics without be an normal error.
