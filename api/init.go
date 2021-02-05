package api

import (
	"net/http"
	"strings"

	"github.com/gobuffalo/packr/v2"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/orange-cloudfoundry/promfetcher/fetchers"
	"github.com/orange-cloudfoundry/promfetcher/userdocs"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Api struct {
	metFetcher *fetchers.MetricsFetcher
}

func Register(rtr *mux.Router, metFetcher *fetchers.MetricsFetcher, broker *Broker, userdocs *userdocs.UserDoc) {
	api := &Api{
		metFetcher: metFetcher,
	}

	handlerMetrics := handlers.CompressHandler(http.HandlerFunc(api.metrics))
	rtr.Handle("/v1/apps/{appIdOrPathOrName:.*}/metrics", handlerMetrics).
		Methods(http.MethodGet)

	rtr.Handle("/v1/apps/metrics", handlerMetrics).
		Methods(http.MethodGet)

	handlerOnlyAppMetrics := handlers.CompressHandler(forceOnlyForApp(http.HandlerFunc(api.metrics)))

	rtr.Handle("/v1/apps/{appIdOrPathOrName:.*}/only-app-metrics", handlerOnlyAppMetrics).
		Methods(http.MethodGet)

	rtr.Handle("/v1/apps/only-app-metrics", handlerOnlyAppMetrics).
		Methods(http.MethodGet)

	rtr.NewRoute().MatcherFunc(func(req *http.Request, m *mux.RouteMatch) bool {
		return strings.HasPrefix(req.URL.Path, "/broker/v2")
	}).Handler(http.StripPrefix("/broker", broker.Handler()))

	boxAsset := packr.New("userdocs_assets", "../userdocs/assets")
	rtr.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(boxAsset)))
	rtr.Handle("/doc", userdocs)
	rtr.Handle("/metrics", promhttp.Handler())
}
