package api

import (
	log "github.com/sirupsen/logrus"
	"io/fs"
	"net/http"
	"strings"

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
	handlerOnlyAppMetrics := handlers.CompressHandler(forceOnlyForApp(http.HandlerFunc(api.metrics)))

	// API v1: deprecated
	routerApiV1 := rtr.PathPrefix("/v1").Subrouter()
	routerApiV1.Handle("/apps/{appIdOrPathOrName:.*}/metrics", handlerMetrics).
		Methods(http.MethodGet)

	routerApiV1.Handle("/apps/metrics", handlerMetrics).
		Methods(http.MethodGet)

	routerApiV1.Handle("/apps/{appIdOrPathOrName:.*}/only-app-metrics", handlerOnlyAppMetrics).
		Methods(http.MethodGet)

	routerApiV1.Handle("/apps/only-app-metrics", handlerOnlyAppMetrics).
		Methods(http.MethodGet)

	// API v2
	routerApiV2 := rtr.PathPrefix("/v2").Subrouter()
	routerApiV2.Handle("/apps/{appIdOrPathOrName:.*}/metrics", handlerMetrics).
		Methods(http.MethodGet)

	routerApiV2.Handle("/apps/metrics", handlerMetrics).
		Methods(http.MethodGet)

	routerApiV2.Handle("/apps/{appIdOrPathOrName:.*}/only-app-metrics", handlerOnlyAppMetrics).
		Methods(http.MethodGet)

	routerApiV2.Handle("/apps/only-app-metrics", handlerOnlyAppMetrics).
		Methods(http.MethodGet)

	// non-API routes
	rtr.NewRoute().MatcherFunc(func(req *http.Request, m *mux.RouteMatch) bool {
		return strings.HasPrefix(req.URL.Path, "/broker/v2")
	}).Handler(http.StripPrefix("/broker", broker.Handler()))

	htmlContent, err := fs.Sub(userdocs.EmbededUserDoc, "assets")
	if err != nil {
		log.Fatal(err)
	}
	rtr.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(http.FS(htmlContent))))
	rtr.Handle("/doc", userdocs)
	rtr.Handle("/metrics", promhttp.Handler())
}
