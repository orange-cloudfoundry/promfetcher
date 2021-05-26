## If metrics available on `/metrics` on your app

You have nothing to do, you can retrieve app instances metrics by simply call one of:

- [{{.BaseURL}}/v1/apps/\[org_name\]/\[space_name\]/\[app_name\]/metrics]({{.BaseURL}}/v1/apps/{org_name}/{space_name}/{app_name}/metrics)
- [{{.BaseURL}}/v1/apps/\[app_id\]/metrics]({{.BaseURL}}/v1/apps/{app_id}/metrics)
- [{{.BaseURL}}/v1/apps/metrics?app="\[org_name\]/\[space_name\]/\[app_name\]"]({{.BaseURL}}/v1/apps/metrics?app="\[org_name\]/\[space_name\]/\[app_name\]")
- [{{.BaseURL}}/v1/apps/metrics?app="\[app_id\]"]({{.BaseURL}}/v1/apps/metrics?app="\[app_id\]")
- [{{.BaseURL}}/v1/apps/\[route.url.com\]/metrics]({{.BaseURL}}/v1/apps/{route.url.com}/metrics)
- [{{.BaseURL}}/v1/apps/metrics?route_url="\[route.url.com\]"]({{.BaseURL}}/v1/apps/metrics?route_url="\[route.url.com\]")

## Set a different endpoint

Add url param `metric_path=/my-metrics/endpoint`, e.g.:

- [{{.BaseURL}}/v1/apps/\[org_name\]/\[space_name\]/\[app_name\]/metrics?metric_path=/my-metrics/endpoint]({{.BaseURL}}/v1/apps/{org_name}/{space_name}/{app_name}/metrics?metric_path=/my-metrics/endpoint)

## Pass http headers to app, useful for authentication

If you do a request with headers, they are all passed to app.

This is useful for authentication purpose, example on basic auth

1. I have an app with metrics on `/metrics` but it is protected with basic auth `foo`/`bar`
2. You can perform curl: `curl https://foo:bar@{{.BaseURL}}/v1/apps/my-app/metrics`
3. Basic auth header are passed to app and you can retrieve information (note that promfetcher do not store anything)

## Retrieving only metrics from your app and not those from external

Use `/only-app-metrics` instead of `/metrics`, e.g.:

- [{{.BaseURL}}/v1/apps/\[org_name\]/\[space_name\]/\[app_name\]/only-app-metrics]({{.BaseURL}}/v1/apps/{org_name}/{space_name}/{app_name}/only-app-metrics)
- [{{.BaseURL}}/v1/apps/only-app-metrics?app="\[app_id\]"]({{.BaseURL}}/v1/apps/only-app-metrics?app="\[app_id\]")
