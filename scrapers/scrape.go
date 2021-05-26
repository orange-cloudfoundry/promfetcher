package scrapers

import (
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/orange-cloudfoundry/promfetcher/clients"
	"github.com/orange-cloudfoundry/promfetcher/errors"
	"github.com/orange-cloudfoundry/promfetcher/models"
)

const acceptHeader = `application/openmetrics-text; version=0.0.1,text/plain;version=0.0.4;q=0.5,*/*;q=0.1`

type Scraper struct {
	backendFactory *clients.BackendFactory
	db             *gorm.DB
	outboundIp     string
}

func NewScraper(backendFactory *clients.BackendFactory, db *gorm.DB) *Scraper {
	return &Scraper{backendFactory: backendFactory, db: db}

}

func (s *Scraper) GetOutboundIP() string {
	if s.outboundIp != "" {
		return s.outboundIp
	}

	// address doesn't need to exists
	conn, err := net.Dial("udp", "10.0.0.1:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	s.outboundIp = localAddr.IP.String()
	return s.outboundIp
}

func (s Scraper) Scrape(route *models.Route, metricPathDefault string, headers http.Header) (io.ReadCloser, error) {
	scheme := "http"
	if route.TLS {
		scheme = "https"
	}
	endpoint := metricPathDefault
	if route.MetricsPath != "" {
		endpoint = route.MetricsPath
	}
	if s.db != nil && route.MetricsPath == "" {
		var appEndpoint models.AppEndpoint
		s.db.First(&appEndpoint, "app_guid = ?", route.Tags.AppID)
		if appEndpoint.GUID != "" {
			endpoint = appEndpoint.Endpoint
		}
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s://%s%s", scheme, route.Address, endpoint), nil)
	if err != nil {
		return nil, err
	}
	if len(headers) > 0 {
		for k, v := range headers {
			req.Header[k] = v
		}
	}
	req.Header.Add("Accept", acceptHeader)
	req.Header.Add("Accept-Encoding", "gzip")
	req.Header.Set("X-Prometheus-Scrape-Timeout-Seconds", fmt.Sprintf("%f", (30*time.Second).Seconds()))
	req.Header.Set("X-Forwarded-Proto", scheme)
	req.Header.Set("X-Promfetcher-Scrapping", "true")
	req.Header.Set("X-Forwarded-For", s.GetOutboundIP())
	req.Host = route.Host
	if route.URLParams != nil && len(route.URLParams) > 0 {
		urlParamsCurrent := req.URL.Query()
		for key, values := range route.URLParams {
			urlParamsCurrent[key] = values
		}
		req.URL.RawQuery = urlParamsCurrent.Encode()
	}
	client := s.backendFactory.NewClient(route)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode >= 400 && resp.StatusCode <= 499 {
			return nil, errors.ErrNoEndpointFound(
				fmt.Sprintf(
					"%s/%s/%s (status code %d)",
					route.Tags.OrganizationName,
					route.Tags.SpaceName,
					route.Tags.AppName,
					resp.StatusCode,
				), endpoint,
			)
		}
		return nil, fmt.Errorf("server returned HTTP status %s", resp.Status)
	}

	if resp.Header.Get("Content-Encoding") != "gzip" {
		return resp.Body, nil
	}
	gzReader, err := NewReaderGzip(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}
	return gzReader, nil
}

type ReaderGzip struct {
	main io.ReadCloser
	gzip *gzip.Reader
}

func NewReaderGzip(main io.ReadCloser) (*ReaderGzip, error) {
	gzReader, err := gzip.NewReader(main)
	if err != nil {
		return nil, err
	}
	return &ReaderGzip{
		main: main,
		gzip: gzReader,
	}, nil
}

func (r ReaderGzip) Read(p []byte) (n int, err error) {
	return r.gzip.Read(p)
}

func (r ReaderGzip) Close() error {
	r.gzip.Close()
	r.main.Close()
	return nil
}
