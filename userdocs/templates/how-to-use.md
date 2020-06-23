## If metrics available on `/metrics` on your app

You have nothing to do, you can retrieve app instances metrics by simply call: 
[{{.BaseURL}}/v1/apps/\[org_name\]/\[space_name\]/\[app_name\]/metrics]({{.BaseURL}}/v1/apps/<org_name>/<space_name>/<app_name>/metrics) or 
[{{.BaseURL}}/v1/apps/\[app_id\]/metrics]({{.BaseURL}}/v1/apps/<app_id>/metrics) or 
[{{.BaseURL}}/v1/apps/metrics?app="\[org_name\]/\[space_name\]/\[app_name\]"]({{.BaseURL}}/v1/apps/metrics?app="\[org_name\]/\[space_name\]/\[app_name\]") or
[{{.BaseURL}}/v1/apps/metrics?app="\[app_id\]"]({{.BaseURL}}/v1/apps/metrics?app="\[app_id\]")


## Set a different endpoint

Simply run create-service command on cf cli and bind it to an app with you personal endpoint:
```bash
$ cf create-service promfetcher fetch-app my-fetcher
$ cf bind-service <my-app> my-fetcher -c '{"endpoint": "/my-metrics/endpoint"}'
```

You will now be able to do what describe in previous section