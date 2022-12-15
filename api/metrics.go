package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	"github.com/orange-cloudfoundry/promfetcher/models"
	"github.com/prometheus/common/expfmt"

	"github.com/orange-cloudfoundry/promfetcher/errors"
)

func (a Api) metrics(w http.ResponseWriter, req *http.Request) {
	// extract the API version from the requested path (ie: /v2)
	// and set it to an HTTP header
	apiVersion := regexp.MustCompile("/v([0-9]+)(?:/|$)").FindStringSubmatch(req.URL.Path)
	if len(apiVersion) == 2 {
		req.Header.Set(models.XPromfetcherApiVersion, apiVersion[1])
	} else {
		// default to v1
		req.Header.Set(models.XPromfetcherApiVersion, "1")
	}

	appIdOrPathOrName, ok := mux.Vars(req)["appIdOrPathOrName"]
	if !ok {
		appIdOrPathOrName = req.URL.Query().Get("app")
	}
	if appIdOrPathOrName == "" {
		appIdOrPathOrName = req.URL.Query().Get("route_url")
	}
	if appIdOrPathOrName == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("%d %s: You must set app id or path", http.StatusBadRequest, http.StatusText(http.StatusBadRequest))))
		return
	}
	metricPathDefault := strings.TrimSpace(req.URL.Query().Get("metric_path"))
	if metricPathDefault == "" {
		metricPathDefault = "/metrics"
	}
	if metricPathDefault[0] != '/' {
		metricPathDefault = "/" + metricPathDefault
	}

	_, onlyAppMetrics := req.URL.Query()["only_from_app"]

	headersMetrics := make(http.Header)
	auth := req.Header.Get("Authorization")
	if auth != "" {
		headersMetrics.Set("Authorization", auth)
	}

	metrics, err := a.metFetcher.Metrics(appIdOrPathOrName, metricPathDefault, onlyAppMetrics, headersMetrics)
	if err != nil {
		if errFetch, ok := err.(*errors.ErrFetch); ok {
			w.WriteHeader(errFetch.Code)
			w.Write([]byte(errFetch.Error()))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("%d %s: %s", http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), err.Error())))
		return
	}
	w.WriteHeader(http.StatusOK)
	for _, metric := range metrics {
		expfmt.MetricFamilyToText(w, metric)
		w.Write([]byte("\n"))
	}
}

func forceOnlyForApp(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		query := req.URL.Query()
		query.Set("only_from_app", "1")
		req.URL.RawQuery = query.Encode()
		next.ServeHTTP(w, req)
	})
}
