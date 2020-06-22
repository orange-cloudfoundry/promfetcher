package fetchers

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/orange-cloudfoundry/promfetcher/config"
	"github.com/orange-cloudfoundry/promfetcher/metrics"
	"github.com/orange-cloudfoundry/promfetcher/models"
	log "github.com/sirupsen/logrus"
)

type RoutesFetcher struct {
	mu              sync.Mutex
	routes          *models.Routes
	httpClient      *http.Client
	lastSuccessTime time.Time
}

func NewRoutesFetcher(confGorouter config.GorouterConfig) *RoutesFetcher {
	rts := make(models.Routes)
	return &RoutesFetcher{
		mu:              sync.Mutex{},
		routes:          &rts,
		lastSuccessTime: time.Now(),
		httpClient: &http.Client{
			Transport: NewFetcherTransport(confGorouter, &http.Transport{
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
		},
	}
}

func (f RoutesFetcher) Run() {
	entry := log.WithField("component", "fetcher")
	go func() {
		for {
			err := f.updateRoutes()
			if err != nil {
				entry.Warnf("Error updating routes: %s", err.Error())
				metrics.ScrapeRouteFailedTotal.With(map[string]string{}).Inc()
			} else {
				f.mu.Lock()
				f.lastSuccessTime = time.Now()
				f.mu.Unlock()
			}
			metrics.LatestScrapeRoute.With(map[string]string{}).Set(time.Now().Sub(f.lastSuccessTime).Seconds())
			time.Sleep(30 * time.Second)
		}
	}()
}

func (f *RoutesFetcher) updateRoutes() error {
	resp, err := f.httpClient.Get("/routes")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var routes models.Routes
	err = json.Unmarshal(b, &routes)
	if err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	*f.routes = routes
	return nil
}

func (f RoutesFetcher) Routes() models.Routes {
	if f.routes == nil {
		return make(models.Routes)
	}
	return *f.routes
}
