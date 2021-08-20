package fetchers

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/orange-cloudfoundry/promfetcher/config"
	"github.com/orange-cloudfoundry/promfetcher/healthchecks"
	"github.com/orange-cloudfoundry/promfetcher/metrics"
	"github.com/orange-cloudfoundry/promfetcher/models"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . RoutesFetch

type RoutesFetch interface {
	Run()
	Routes() models.Routes
}

type RoutesFetcher struct {
	mu              sync.Mutex
	routes          *models.Routes
	goRtrHttpClts   []*http.Client
	lastSuccessTime time.Time
	healthCheck     *healthchecks.HealthCheck
}

func NewRoutesFetcher(confGorouters []config.GorouterConfig, healthCheck *healthchecks.HealthCheck) *RoutesFetcher {
	rts := make(models.Routes)

	goRtrHttpClts := make([]*http.Client, 0)
	for _, conf := range confGorouters {
		goRtrHttpClts = append(goRtrHttpClts, &http.Client{
			Transport: NewFetcherTransport(conf, &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
					DualStack: true,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			}),
			Timeout: 30 * time.Second,
		})
	}
	return &RoutesFetcher{
		mu:              sync.Mutex{},
		routes:          &rts,
		lastSuccessTime: time.Now(),
		goRtrHttpClts:   goRtrHttpClts,
		healthCheck:     healthCheck,
	}
}

func (f *RoutesFetcher) Run() {
	entry := log.WithField("component", "fetcher")
	go func() {
		for {
			err := f.updateRoutes()
			if err != nil {
				entry.Warnf("Error updating routes: %s", err.Error())
				metrics.ScrapeRouteFailedTotal.With(map[string]string{}).Inc()
			} else {
				f.mu.Lock()
				if f.healthCheck.Health() == healthchecks.Initializing {
					f.healthCheck.SetHealth(healthchecks.Healthy)
				}
				f.lastSuccessTime = time.Now()
				f.mu.Unlock()
			}

			metrics.LatestScrapeRoute.With(map[string]string{}).Set(time.Since(f.lastSuccessTime).Seconds())
			time.Sleep(30 * time.Second)
		}
	}()
}

func (f *RoutesFetcher) updateRoutes() error {
	routes := make(models.Routes)

	for _, client := range f.goRtrHttpClts {
		resp, err := client.Get("/routes")
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		var tmpRoutes models.Routes
		err = json.Unmarshal(b, &tmpRoutes)
		if err != nil {
			return err
		}

		// insert host in routes
		for routeName, routesInfo := range tmpRoutes {
			// route can have path inside, we split to get only host
			host := strings.SplitN(routeName, "/", 2)[0]
			for _, route := range routesInfo {
				route.Host = host
			}
		}
		if len(routes) == 0 {
			routes = tmpRoutes
			continue
		}

		for routeName, routesInfo := range tmpRoutes {
			// only add when unknown
			if _, ok := routes[routeName]; ok {
				continue
			}
			routes[routeName] = routesInfo
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	*f.routes = routes
	return nil
}

func (f *RoutesFetcher) Routes() models.Routes {
	if f.routes == nil {
		return make(models.Routes)
	}
	return *f.routes
}
