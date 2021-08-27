package scrapers_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/jinzhu/gorm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/orange-cloudfoundry/promfetcher/clients"
	"github.com/orange-cloudfoundry/promfetcher/config"
	"github.com/orange-cloudfoundry/promfetcher/models"
	"github.com/orange-cloudfoundry/promfetcher/scrapers"
)

var _ = Describe("Scraper", func() {
	var db *gorm.DB
	var err error
	var scraper *scrapers.Scraper
	var server *ghttp.Server

	BeforeEach(func() {
		db, err = gorm.Open("sqlite3", "file::memory:?cache=shared")
		Expect(err).ShouldNot(HaveOccurred())

		db.AutoMigrate(&models.AppEndpoint{})
		Expect(err).ShouldNot(HaveOccurred())

		app := models.AppEndpoint{
			AppGUID:  "d245c244-1875-a718-1248-2547e141a45c",
			Endpoint: "/metrics",
		}

		db.Create(&app)

		c, err := config.DefaultConfig()
		Expect(err).ShouldNot(HaveOccurred())

		backendFactory := clients.NewBackendFactory(*c)
		scraper = scrapers.NewScraper(backendFactory, db)

		server = ghttp.NewServer()
	})

	AfterEach(func() {
		server.Close()
	})

	Context("Scrape", func() {
		var serverURL *url.URL
		var content = "test_scrape_error 0"
		BeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", fmt.Sprintf("/metrics")),
					ghttp.RespondWith(http.StatusOK, content),
				),
			)
			serverURL, err = url.Parse(server.URL())
			Expect(err).ToNot(HaveOccurred())
		})

		It("scrapes metrics from an app", func() {
			route := &models.Route{
				PrivateInstanceID: "a758f25d-2d01-419e-b63b-de3aabcd9e15",
				Address:           serverURL.Host,
				TLS:               false,
				MetricsPath:       "/metrics",
			}

			resp, err := scraper.Scrape(route, "", http.Header{})
			Expect(err).ShouldNot(HaveOccurred())
			defer resp.Close()

			body, err := ioutil.ReadAll(resp)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(body)).To(Equal(content))
		})
	})

	Context("GetOutboundIP", func() {
		It("gets local ip", func() {
			ip := scraper.GetOutboundIP()
			Expect(ip).Should(MatchRegexp(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`))

		})
	})
})
