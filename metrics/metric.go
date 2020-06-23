package metrics

import (
	"github.com/orange-cloudfoundry/promfetcher/models"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	MetricFetchFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "metric_fetch_failed_total",
			Help: "Number of non fetched metrics without be an normal error.",
		},
		[]string{"organization_id", "space_id", "app_id", "organization_name", "space_name", "app_name", "index", "instance_id"},
	)
	LatestScrapeRoute = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "latest_time_scrape_route",
			Help: "Last time that route has been scraped in seconds.",
		},
		[]string{},
	)
	ScrapeRouteFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scrape_route_failed_total",
			Help: "Number of non fetched metrics without be an normal error.",
		},
		[]string{},
	)
)

func RouteToLabel(route models.Route) prometheus.Labels {
	return map[string]string{
		"organization_id":   route.Tags.OrganizationID,
		"space_id":          route.Tags.SpaceID,
		"app_id":            route.Tags.AppID,
		"organization_name": route.Tags.OrganizationName,
		"space_name":        route.Tags.SpaceName,
		"app_name":          route.Tags.AppName,
		"index":             route.Tags.InstanceID,
		"instance_id":       route.Tags.InstanceID,
	}
}

func init() {
	prometheus.MustRegister(MetricFetchFailedTotal)
	prometheus.MustRegister(LatestScrapeRoute)
	prometheus.MustRegister(ScrapeRouteFailedTotal)
}
