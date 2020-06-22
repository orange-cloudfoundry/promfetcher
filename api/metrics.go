package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/orange-cloudfoundry/promfetcher/errors"
	"github.com/prometheus/common/expfmt"
)

func (a Api) metrics(w http.ResponseWriter, req *http.Request) {
	appIdOrPath := mux.Vars(req)["appIdOrPath"]
	metrics, err := a.metFetcher.Metrics(appIdOrPath)
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
