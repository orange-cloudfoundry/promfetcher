package api_test

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/jinzhu/gorm"
	"github.com/orange-cloudfoundry/promfetcher/api"
	"github.com/orange-cloudfoundry/promfetcher/config"
	"github.com/orange-cloudfoundry/promfetcher/models"
	"github.com/pivotal-cf/brokerapi/v7/domain"
)

var _ = Describe("Api/Broker", func() {
	var db *gorm.DB
	var err error
	var broker *api.Broker
	var router *mux.Router

	BeforeEach(func() {
		db, err = gorm.Open("sqlite3", "file::memory:?cache=shared")
		Expect(err).ShouldNot(HaveOccurred())

		db.AutoMigrate(&models.AppEndpoint{})
		Expect(err).ShouldNot(HaveOccurred())

		broker = api.NewBroker(
			config.BrokerConfig{
				BrokerPlanID:    "e2900be3-709b-419e-b63b-de3aabcd9e15",
				BrokerServiceID: "75bcebab-cc25-4ef6-89dc-a91b953919f1",
				User:            "user",
				Pass:            "password",
			},
			"http://localhost:8085",
			db,
		)

		router = broker.Handler().(*mux.Router)
	})

	AfterEach(func() {
		err = db.Close()
		Expect(err).ShouldNot(HaveOccurred())
	})

	Context("Service", func() {
		It("returns service informations", func() {
			services, err := broker.Services(nil)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(services).To(HaveLen(1))

			jsonString := `{
				"id": "75bcebab-cc25-4ef6-89dc-a91b953919f1",
				"name": "promfetcher",
				"description": "Fetch your prometheus metrics on each instance of your app",
				"bindable": true,
				"bindings_retrievable": true,
				"plan_updateable": false,
				"plans": [
					{
						"id": "e2900be3-709b-419e-b63b-de3aabcd9e15",
						"name": "fetch-app",
						"description": "Fetch your prometheus metrics on each instance of your app by setting an endpoint to scrap",
						"free": true,
						"bindable": true,
						"metadata": {
							"displayName": "fetch-app"
						}
					}
				],
				"metadata": {
					"displayName": "promfetcher",
					"documentationUrl": "http://localhost:8085/doc",
					"longDescription": "Fetch your prometheus metrics on each instance of your app.",
					"providerDisplayName": "Orange"
				}
			}`

			Expect(json.Marshal(services[0])).To(MatchJSON(jsonString))
		})
	})

	Context("Bindings", func() {
		var instanceID = "a758f25d-2d01-419e-b63b-de3aabcd9e15"
		var bindingID = "abcdb63b-b63b-2d01-873b-e3a758f25d48"
		var app = models.AppEndpoint{}

		It("(un)binds service instance", func() {
			var details = domain.BindDetails{
				AppGUID: "d245c244-1875-a718-1248-2547e141a45c",
				RawParameters: []byte(`{
					"endpoint": "/metrics"
				}`),
			}

			_, err := broker.Bind(nil, instanceID, bindingID, details, false)
			Expect(err).ShouldNot(HaveOccurred())

			result := db.First(&app, "guid = ?", bindingID)
			Expect(result.RowsAffected).Should(BeEquivalentTo(1))
			Expect(app.GUID).To(Equal(bindingID))
			Expect(app.AppGUID).To(Equal(details.AppGUID))
			Expect(app.Endpoint).To(Equal("/metrics"))

			bindingSpec, err := broker.GetBinding(nil, instanceID, bindingID)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(bindingSpec).To(Equal(domain.GetBindingSpec{Credentials: map[string]string{"endpoint": "/metrics"}}))

			var unbindDetails = domain.UnbindDetails{
				PlanID:    details.PlanID,
				ServiceID: details.ServiceID,
			}
			broker.Unbind(nil, instanceID, bindingID, unbindDetails, false)
			result = db.First(&app, "guid = ?", bindingID)
			Expect(result.RowsAffected).Should(BeZero())
		})

		It("fail gracefully when not found", func() {
			bindingSpec, err := broker.GetBinding(nil, instanceID, bindingID)
			Expect(bindingSpec).To(Equal(domain.GetBindingSpec{}))
			Expect(err).To(HaveOccurred())
		})

	})

	Context("Handler", func() {
		var routeMatch = mux.RouteMatch{}
		var url = url.URL{Path: "/v2/catalog"}
		var request = http.Request{Method: "GET", URL: &url}

		It("fails when /v2/catalog route is not found", func() {
			Expect(router.Match(&request, &routeMatch)).To(BeTrue())
		})
	})
})
