package api

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func AccessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.WithFields(log.Fields{
			"method": req.Method,
			"uri":    req.URL.RequestURI(),
		}).Info("access")

		next.ServeHTTP(w, req)
	})
}
