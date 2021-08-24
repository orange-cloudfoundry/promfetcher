package api_test

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi/v7/domain"

	"github.com/jinzhu/gorm"
	"github.com/orange-cloudfoundry/promfetcher/api"
	"github.com/orange-cloudfoundry/promfetcher/config"
	"github.com/orange-cloudfoundry/promfetcher/models"
)

var _ = Describe("Api/Broker", func() {
	var db *gorm.DB
	//var sdb *sql.DB
	//var mock sqlmock.Sqlmock
	var err error
	var broker *api.Broker

	BeforeEach(func() {
		/*sdb, mock, err = sqlmock.New()
		Expect(err).ShouldNot(HaveOccurred())

		db, err = gorm.Open("sqlite3", sdb)
		Expect(err).ShouldNot(HaveOccurred())*/

		db, err = gorm.Open("sqlite3", "file::memory:?cache=shared")
		Expect(err).ShouldNot(HaveOccurred())

		db.AutoMigrate(&models.AppEndpoint{})

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
	})

	AfterEach(func() {
		db.Close()
		Expect(err).ShouldNot(HaveOccurred())
	})

	Context("Service", func() {
		It("returns service informations", func() {
			var ctx = context.TODO()
			services, err := broker.Services(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			txt, err := json.Marshal(services[0])
			fmt.Println("test:", string(txt))

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

	var instanceID = "a758f25d-2d01-419e-b63b-de3aabcd9e15"
	var bindingID = "abcdb63b-b63b-2d01-873b-e3a758f25d48"
	var details = domain.BindDetails{
		AppGUID: "d245c244-1875-a718-1248-2547e141a45c",
		RawParameters: []byte(`{
			"endpoint": "/metrics"
		}`),
	}

	Context("Bind", func() {
		It("binds service instance", func() {
			_, err := broker.Bind(nil, instanceID, bindingID, details, false)
			Expect(err).ShouldNot(HaveOccurred())

			var app = models.AppEndpoint{}
			db.First(&app, "guid = ?", bindingID)
			Expect(app.GUID).To(Equal(bindingID))
			Expect(app.AppGUID).To(Equal(details.AppGUID))
			Expect(app.Endpoint).To(Equal("/metrics"))
		})
	})

})
