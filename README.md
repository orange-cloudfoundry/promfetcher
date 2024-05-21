# Promfetcher

Promfetcher was made for [Cloud Foundry] in order to expose [OpenMetrics] from all instances
of an App in a Cloud Foundry environment.

User can retrieve the metrics by simply calling `/v1/apps/${org_name}/${space_name}/${app_name}/metrics`,
or by the application route URL through `/v1/apps/metrics?route_url=my.route.com`,
which will merge all the metrics from an App instances,
then add the following labels (similar to what can be found in the variable [`VCAP_APPLICATION`]):

- `organization_id` - The GUID identifying the org where the app is deployed.
- `organization_name` - The human-readable name of the org where the app is deployed.
- `space_id` - The GUID identifying the space where the app is deployed.
- `space_name` - The human-readable name of the space where the app is deployed.
- `app_id` - The GUID identifying the app.
- `app_name` - The name assigned to the app when it was pushed.
- `index` - The index number of the app instance.
- `instance_id` - (same as the `index`)
- `instance` - The real IP address and port of the container running the App instance.

It is also a Cloud Foundry [service broker] able to expose an endpoint containing some system metrics
for applications without one.

[Cloud Foundry]: https://cloudfoundry.org
[service broker]: https://docs.cloudfoundry.org/services/overview.html
[`VCAP_APPLICATION`]: https://docs.cloudfoundry.org/devguide/deploy-apps/environment-variable.html#VCAP-APPLICATION

## Usage

### Set up

On Cloud Foundry, you should deploy it through its corresponding [BOSH release]:
https://github.com/orange-cloudfoundry/promfetcher-release

[BOSH release]: https://bosh.io/releases/

### Standard endpoint

If your Apps metrics are available on the `/metrics` path (as per [OpenMetrics] recommendations),
you have nothing else to do and you can retrieve App instances metrics by simply calling one of:

- `promfetcher.example.net/v1/apps/{org_name}/{space_name}/{app_name}/metrics`
- `promfetcher.example.net/v1/apps/{app_id}/metrics`
- `promfetcher.example.net/v1/apps/metrics?app="[org_name]/[space_name]/[app_name]"`
- `promfetcher.example.net/v1/apps/metrics?app="[app_id]"`
- `promfetcher.example.net/v1/apps/{route.url.com}/metrics`
- `promfetcher.example.net/v1/apps/metrics?route_url="[route.url.com]"`

To retrieve only the metrics exposed from your application (without the Promfetcher sugar coating),
use `/only-app-metrics` instead of `/metrics`, i.e.:

- `promfetcher.example.net/v1/apps/{org_name}/{space_name}/{app_name}/only-app-metrics`
- `promfetcher.example.net/v1/apps/only-app-metrics?app="[app_id]"`

### Setting a custom endpoint

Add to your querystring the parameter `metric_path={/my-metrics/endpoint}`, i.e.:

- `promfetcher.example.net/v1/apps/{org_name}/{space_name}/{app_name}/metrics?metric_path=/my-metrics/endpoint`

### Pass HTTP headers to the App

If you do a request with headers, they are all passed to the App.

This is useful for authentication purpose, for example:

1. I have an App with metrics on `/metrics` which is protected with HTTP Basic Auth
2. You can perform curl: `curl https://username:password@promfetcher.example.net/v1/apps/my-app/metrics`
3. HTTP Basic Auth headers are passed to the App, and you can retrieve the information
   (note that Promfetcher does not store any data)

## Under the hood

### How does it work?

Promfetcher only needs [NATS](https://nats.io/) an infrastructure that allows data exchange, segmented in the form of messages (We call this a "message oriented middleware"). Promfetcher subscribes to NATS in order to receive messages about routes and then manages its own routing table in memory.

When asking metrics for an App, Promfetcher will asynchronously call all App instances
(provided by the gorouter routing table) metrics endpoint and merge them together with new labels.

Example, given an App with metrics from instance 0:

```
go_memstats_mspan_sys_bytes{} 65536
```

And metrics from App instance 1:

```
go_memstats_mspan_sys_bytes{} 5600
```

Promfetcher will merge them to:

```
go_memstats_mspan_sys_bytes{organization_id="7d66c7e7-196a-40e5-a259-f5afaf6a56f4",space_id="2ac205af-e18f-49a9-9a8b-48ef2bab2292",app_id="621617db-9dd9-4211-8848-b245f3ea16b2",organization_name="system",space_name="tools",app_name="app",index="0",instance_id="0",instance="172.76.112.90:61038"} 65536
go_memstats_mspan_sys_bytes{organization_id="7d66c7e7-196a-40e5-a259-f5afaf6a56f4",space_id="2ac205af-e18f-49a9-9a8b-48ef2bab2292",app_id="621617db-9dd9-4211-8848-b245f3ea16b2",organization_name="system",space_name="tools",app_name="app",index="1",instance_id="1",instance="172.76.112.91:61010"} 65536
```


### Graceful shutdown

Upon receiving `SIGINT`, `SIGTERM` or `SIGUSR1`, Promfetcher will stop listening to new connections
and will wait up to 15 seconds to let the processing transactions a chance to finish before exiting.

### Health Check

The default Promfetcher [Health Check] is of "port" type on `8080`.

Promfetcher answers with an HTTP 200 status if healthy and HTTP 503 otherwise.

[Health Check]: https://docs.cloudfoundry.org/devguide/deploy-apps/healthchecks.html

The administrator can send a `SIGUSR1` to force an unhealthy status in addition to stop it gracefully.

### Promfetcher's internal metrics

Promfetcher metrics are exposed on the [OpenMetrics] standard path `/metrics` and contains the following:

[//]: # (curl -s https://promfetcher.example.net/metrics | sed -n 's/^# HELP \&#40;promfetch_[^ ]*\&#41;/- `\1`:/p')

- `promfetch_metric_fetch_failed_total`: Number of non-fetched metrics without be a normal error.
- `promfetch_metric_fetch_success_total`: Number of fetched metrics succeeded for an App (App instances calls are summed).
- `promfetch_latest_time_scrape_route`: Last time that route has been scraped, in seconds.
- `promfetch_scrape_route_failed_total`: Number of non-fetched metrics without be an normal error.

[OpenMetrics]: https://github.com/OpenObservability/OpenMetrics/blob/v1.0.0/specification/OpenMetrics.md
