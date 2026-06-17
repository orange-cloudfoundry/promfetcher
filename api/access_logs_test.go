package api_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/orange-cloudfoundry/promfetcher/api"
	log "github.com/sirupsen/logrus"
)

var _ = Describe("Api/AccessLogs", func() {
	It("logs called URI including query string", func() {
		buf := &bytes.Buffer{}
		logger := log.StandardLogger()
		originalOut := logger.Out
		originalFormatter := logger.Formatter
		originalLevel := logger.GetLevel()
		defer func() {
			log.SetOutput(originalOut)
			log.SetFormatter(originalFormatter)
			log.SetLevel(originalLevel)
		}()

		log.SetOutput(buf)
		log.SetFormatter(&log.TextFormatter{DisableTimestamp: true, DisableColors: true})
		log.SetLevel(log.InfoLevel)

		handler := api.AccessLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))

		req := httptest.NewRequest(http.MethodGet, "/v2/apps/metrics?app=my-app&metric_path=/custom", nil)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)

		Expect(resp.Code).To(Equal(http.StatusNoContent))
		Expect(buf.String()).To(ContainSubstring("uri=\"/v2/apps/metrics?app=my-app&metric_path=/custom\""))
	})
})

