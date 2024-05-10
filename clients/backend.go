package clients

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/orange-cloudfoundry/promfetcher/config"
	"github.com/orange-cloudfoundry/promfetcher/models"
)

type BackendFactory struct {
	factory FactoryRoundTripper
}

func NewBackendFactory(c config.Config) *BackendFactory {
	backendTLSConfig := &tls.Config{
		InsecureSkipVerify: c.SkipSSLValidation,
		RootCAs:            c.CAPool,
		Certificates:       []tls.Certificate{c.Backends.ClientAuthCertificate},
	}
	return &BackendFactory{
		factory: FactoryRoundTripper{
			Template: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				DisableKeepAlives:   c.DisableKeepAlives,
				MaxIdleConns:        c.MaxIdleConns,
				IdleConnTimeout:     90 * time.Second, // setting the value to golang default transport
				MaxIdleConnsPerHost: c.MaxIdleConnsPerHost,
				DisableCompression:  false,
				TLSClientConfig:     backendTLSConfig,
			},
		},
	}
}

func (f BackendFactory) NewClient(route *models.Route, followRedirect bool) *http.Client {
	client := &http.Client{
		Transport: f.factory.New(route.ServerCertDomainSan),
		Timeout:   30 * time.Second,
	}

	if !followRedirect {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) == 0 {
				return fmt.Errorf("empty previous request for redirect %s, should not happen", req.URL)
			}

			if req.URL.Host == via[0].URL.Host {
				// legitimate redirect (like relative redirect) then continue (just for one hop)
				return nil
			}

			// redirect send outside of cloudfoundry : don't follow
			return errors.New("external redirect detected or too many redirect: give metric_path parameter with direct endpoint to app metrics")
		}
	}

	return client
}
