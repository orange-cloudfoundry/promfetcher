package clients_test

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/orange-cloudfoundry/promfetcher/clients"
	"github.com/orange-cloudfoundry/promfetcher/config"
	"github.com/orange-cloudfoundry/promfetcher/models"
)

var _ = Describe("Backend", func() {
	Context("NewClient", func() {
		It("will give you a http client with given template", func() {
			factory := clients.NewBackendFactory(config.Config{
				CAPool:            &x509.CertPool{},
				SkipSSLValidation: false,
				Backends: config.BackendConfig{
					ClientAuthCertificate: tls.Certificate{},
				},
				DisableKeepAlives:   false,
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
			})

			client := factory.NewClient(&models.Route{ServerCertDomainSan: "my.san"}, false)

			Expect(client).ToNot(BeNil())
			Expect(client.Timeout).To(Equal(30 * time.Second))
			Expect(client.CheckRedirect).ToNot(BeNil())

			httpTrans, ok := client.Transport.(*http.Transport)
			Expect(ok).To(BeTrue())
			Expect(httpTrans.TLSClientConfig.ServerName).To(Equal("my.san"))
			Expect(httpTrans.MaxIdleConns).To(Equal(100))
			Expect(httpTrans.MaxIdleConnsPerHost).To(Equal(100))
		})
	})
})
