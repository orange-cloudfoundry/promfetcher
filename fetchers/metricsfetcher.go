package fetchers

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	log "github.com/sirupsen/logrus"

	"github.com/orange-cloudfoundry/promfetcher/config"
	"github.com/orange-cloudfoundry/promfetcher/errors"
	"github.com/orange-cloudfoundry/promfetcher/metrics"
	"github.com/orange-cloudfoundry/promfetcher/models"
	"github.com/orange-cloudfoundry/promfetcher/scrapers"
)

func ptrString(v string) *string {
	return &v
}

type MetricsFetcher struct {
	scraper           *scrapers.Scraper
	routesFetcher     RoutesFetch
	externalExporters config.ExternalExporters
}

func NewMetricsFetcher(scraper *scrapers.Scraper, routesFetcher RoutesFetch, externalExporters config.ExternalExporters) *MetricsFetcher {
	return &MetricsFetcher{
		scraper:           scraper,
		routesFetcher:     routesFetcher,
		externalExporters: externalExporters,
	}
}

func (f MetricsFetcher) Metrics(appIdOrPathOrName, metricPathDefault string, onlyAppMetrics bool, headers http.Header) (map[string]*dto.MetricFamily, error) {

	routes := f.routesFetcher.Routes().Find(appIdOrPathOrName)
	if len(routes) == 0 {
		return make(map[string]*dto.MetricFamily), errors.ErrNoAppFound(appIdOrPathOrName)
	}
	mapTagsRoute := make(map[string]models.Tags)
	for _, rte := range routes {
		mapTagsRoute[rte.Tags.AppID] = rte.Tags
	}
	jobs := make(chan *models.Route, len(routes))
	errFetch := &errors.ErrFetch{}
	wg := &sync.WaitGroup{}

	muWrite := sync.Mutex{}
	metricsUnmerged := make([]map[string]*dto.MetricFamily, 0)

	if !onlyAppMetrics && f.externalExporters != nil && len(f.externalExporters) > 0 {
		for _, tagRte := range mapTagsRoute {
			tags := models.Tags{
				ProcessType:      "external_exporter",
				Component:        "promfetcher",
				SpaceName:        tagRte.SpaceName,
				OrganizationID:   tagRte.OrganizationID,
				OrganizationName: tagRte.OrganizationName,
				SourceID:         tagRte.SourceID,
				AppID:            tagRte.AppID,
				AppName:          tagRte.AppName,
				SpaceID:          tagRte.SpaceID,
			}
			for _, ee := range f.externalExporters {
				routeExternalExporter, err := ee.ToRoute(tags)
				if err != nil {
					err = fmt.Errorf("error when setting external exporters routes: %s", err.Error())
					newMetrics := f.scrapeExternalExporterError(tags, ee, err)
					metricsUnmerged = append(metricsUnmerged, newMetrics)
					log.WithField("external_exporter", ee.Name).
						WithField("action", "route convert").
						WithField("app", fmt.Sprintf("%s/%s/%s", tags.OrganizationName, tags.SpaceName, tags.AppName)).
						Warningf(err.Error())
					continue
				}
				routes = append(routes, routeExternalExporter)
			}
		}
	}

	wg.Add(len(routes))
	for w := 1; w <= 5; w++ {
		go func(jobs <-chan *models.Route, errFetch *errors.ErrFetch, headers http.Header) {
			for j := range jobs {
				if j.Tags.ProcessType == "external_exporter" {
					headers = nil
				}
				newMetrics, err := f.Metric(j, metricPathDefault, headers)
				if err != nil {
					if errF, ok := err.(*errors.ErrFetch); ok && (f.externalExporters == nil || len(f.externalExporters) == 0) {
						muWrite.Lock()
						*errFetch = *errF
						muWrite.Unlock()
						wg.Done()
						continue
					}
					log.Debugf("Cannot get metric for instance %s for instance id %s (%s/%s/%s) : %s", j.Address, j.Tags.InstanceID, j.Tags.OrganizationName, j.Tags.SpaceName, j.Tags.AppName, err)
					newMetrics = f.scrapeError(j, err)
					metrics.MetricFetchFailedTotal.With(metrics.RouteToLabel(j)).Inc()
				} else {
					metrics.MetricFetchSuccessTotal.With(metrics.RouteToLabelNoInstance(j)).Inc()
				}
				muWrite.Lock()
				metricsUnmerged = append(metricsUnmerged, newMetrics)
				muWrite.Unlock()
				wg.Done()
			}
		}(jobs, errFetch, headers)
	}
	for _, route := range routes {
		jobs <- route
	}
	wg.Wait()
	close(jobs)
	if errFetch.Code != 0 {
		return make(map[string]*dto.MetricFamily), errFetch
	}

	if len(metricsUnmerged) == 0 {
		return make(map[string]*dto.MetricFamily), nil
	}

	base := metricsUnmerged[0]
	for _, metricKV := range metricsUnmerged[1:] {
		for k, metricFamily := range metricKV {
			baseMetricFamily, ok := base[k]
			if !ok {
				base[k] = metricFamily
				continue
			}
			baseMetricFamily.Metric = append(baseMetricFamily.Metric, metricFamily.Metric...)
		}
	}
	return base, nil
}

func (f MetricsFetcher) Metric(route *models.Route, metricPathDefault string, headers http.Header) (map[string]*dto.MetricFamily, error) {
	reader, err := f.scraper.Scrape(route, metricPathDefault, headers)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	parser := &expfmt.TextParser{}
	metricsGroup, err := parser.TextToMetricFamilies(reader)
	if err != nil {
		return nil, err
	}

	for _, metricGroup := range metricsGroup {
		for _, metric := range metricGroup.Metric {
			metric.Label = f.cleanMetricLabels(
				metric.Label,
				"organization_id", "space_id", "app_id",
				"organization_name", "space_name", "app_name",
				"index", "instance_id", "instance",
			)
			metric.Label = append(metric.Label,
				&dto.LabelPair{
					Name:  ptrString("organization_id"),
					Value: ptrString(route.Tags.OrganizationID),
				},
				&dto.LabelPair{
					Name:  ptrString("space_id"),
					Value: ptrString(route.Tags.SpaceID),
				},
				&dto.LabelPair{
					Name:  ptrString("app_id"),
					Value: ptrString(route.Tags.AppID),
				},
				&dto.LabelPair{
					Name:  ptrString("organization_name"),
					Value: ptrString(route.Tags.OrganizationName),
				},
				&dto.LabelPair{
					Name:  ptrString("space_name"),
					Value: ptrString(route.Tags.SpaceName),
				},
				&dto.LabelPair{
					Name:  ptrString("app_name"),
					Value: ptrString(route.Tags.AppName),
				},
			)
			if route.Tags.InstanceID != "" {
				metric.Label = append(metric.Label,
					&dto.LabelPair{
						Name:  ptrString("index"),
						Value: ptrString(route.Tags.InstanceID),
					},
					&dto.LabelPair{
						Name:  ptrString("instance_id"),
						Value: ptrString(route.Tags.InstanceID),
					},
					&dto.LabelPair{
						Name:  ptrString("instance"),
						Value: ptrString(route.Address),
					},
				)
			}

		}
	}
	return metricsGroup, nil
}

func (f MetricsFetcher) cleanMetricLabels(labels []*dto.LabelPair, names ...string) []*dto.LabelPair {
	finalLabels := make([]*dto.LabelPair, 0)
	for _, label := range labels {
		toAdd := true
		for _, name := range names {
			if label.Name != nil && *label.Name == name {
				toAdd = false
				break
			}
		}
		if toAdd {
			finalLabels = append(finalLabels, label)
		}
	}
	return finalLabels
}

func (f MetricsFetcher) scrapeError(route *models.Route, err error) map[string]*dto.MetricFamily {
	name := "promfetcher_scrape_error"
	help := "Promfetcher scrap error on your instance"
	metric := prometheus.NewCounter(prometheus.CounterOpts{
		Name: name,
		Help: help,
		ConstLabels: prometheus.Labels{
			"organization_id":   route.Tags.OrganizationID,
			"space_id":          route.Tags.SpaceID,
			"app_id":            route.Tags.AppID,
			"organization_name": route.Tags.OrganizationName,
			"space_name":        route.Tags.SpaceName,
			"app_name":          route.Tags.AppName,
			"index":             route.Tags.InstanceID,
			"instance_id":       route.Tags.InstanceID,
			"instance":          route.Address,
			"error":             err.Error(),
		},
	})
	metric.Inc()
	var dtoMetric dto.Metric
	metric.Write(&dtoMetric)
	metricType := dto.MetricType_COUNTER
	return map[string]*dto.MetricFamily{
		"promfetcher_scrape_error": {
			Name:   ptrString(name),
			Help:   ptrString(help),
			Type:   &metricType,
			Metric: []*dto.Metric{&dtoMetric},
		},
	}
}

func (f MetricsFetcher) scrapeExternalExporterError(tags models.Tags, externalExporter *config.ExternalExporter, err error) map[string]*dto.MetricFamily {
	name := "promfetcher_scrape_external_exporter_error"
	help := "Promfetcher scrap external exporter error on your instance"
	metric := prometheus.NewCounter(prometheus.CounterOpts{
		Name: name,
		Help: help,
		ConstLabels: prometheus.Labels{
			"organization_id":   tags.OrganizationID,
			"space_id":          tags.SpaceID,
			"app_id":            tags.AppID,
			"organization_name": tags.OrganizationName,
			"space_name":        tags.SpaceName,
			"app_name":          tags.AppName,
			"index":             tags.InstanceID,
			"instance_id":       tags.InstanceID,
			"instance":          externalExporter.Host,
			"name":              externalExporter.Name,
			"error":             err.Error(),
		},
	})
	metric.Inc()
	var dtoMetric dto.Metric
	metric.Write(&dtoMetric)
	metricType := dto.MetricType_COUNTER
	return map[string]*dto.MetricFamily{
		"promfetcher_scrape_external_exporter_error": {
			Name:   ptrString(name),
			Help:   ptrString(help),
			Type:   &metricType,
			Metric: []*dto.Metric{&dtoMetric},
		},
	}
}
