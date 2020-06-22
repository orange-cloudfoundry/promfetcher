package fetchers

import (
	"fmt"
	"net/http"

	"github.com/orange-cloudfoundry/promfetcher/config"
)

type FetcherTransport struct {
	GorouterConf  config.GorouterConfig
	WrapTransport http.RoundTripper
}

func NewFetcherTransport(gorouterConf config.GorouterConfig, transport http.RoundTripper) *FetcherTransport {
	return &FetcherTransport{
		WrapTransport: transport,
		GorouterConf:  gorouterConf,
	}
}

func (t FetcherTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(t.GorouterConf.User, t.GorouterConf.Pass)
	req.URL.Host = fmt.Sprintf("%s:%d", t.GorouterConf.Host, t.GorouterConf.Port)
	req.URL.Scheme = "http"
	return t.WrapTransport.RoundTrip(req)
}
