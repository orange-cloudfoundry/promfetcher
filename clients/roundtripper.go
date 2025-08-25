package clients

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

type FactoryRoundTripper struct {
	Template *http.Transport
}

func (t *FactoryRoundTripper) New(expectedServerName string) http.RoundTripper {

	customTLSConfig := TLSConfigWithServerName(expectedServerName, t.Template.TLSClientConfig)
	maxIdle := 100
	if t.Template.MaxIdleConns != 0 {
		maxIdle = t.Template.MaxIdleConns
	}
	idleConnTimeout := 90 * time.Second
	if t.Template.IdleConnTimeout != 0 {
		idleConnTimeout = t.Template.IdleConnTimeout
	}

	dialContext := (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext
	if t.Template.DialContext != nil {
		dialContext = t.Template.DialContext
	}

	disableKeepAlives := t.Template.DisableKeepAlives

	disableCompression := t.Template.DisableCompression

	maxIdleConnsPerHost := 0
	if t.Template.MaxIdleConnsPerHost != 0 {
		maxIdleConnsPerHost = t.Template.MaxIdleConnsPerHost
	}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          maxIdle,
		IdleConnTimeout:       idleConnTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       customTLSConfig,
		DisableKeepAlives:     disableKeepAlives,
		DisableCompression:    disableCompression,
		MaxIdleConnsPerHost:   maxIdleConnsPerHost,
	}
}

func TLSConfigWithServerName(newServerName string, template *tls.Config) *tls.Config {
	return &tls.Config{
		CipherSuites:       template.CipherSuites,
		InsecureSkipVerify: template.InsecureSkipVerify,
		RootCAs:            template.RootCAs,
		ServerName:         newServerName,
		Certificates:       template.Certificates,
	}
}
