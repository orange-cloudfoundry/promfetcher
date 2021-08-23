package api_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jinzhu/gorm"
	"github.com/orange-cloudfoundry/promfetcher/api"
	"github.com/orange-cloudfoundry/promfetcher/config"
)

var _ = Describe("Api/Broker", func() {
	var db *gorm.DB
	var sdb *sql.DB
	var mock sqlmock.Sqlmock
	var err error
	var broker *api.Broker

	BeforeEach(func() {
		sdb, mock, err = sqlmock.New()
		Expect(err).ShouldNot(HaveOccurred())

		db, err = gorm.Open("sqlite3", sdb)
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
	})

	AfterEach(func() {
		db.Close()
		err := mock.ExpectationsWereMet()
		Expect(err).ShouldNot(HaveOccurred())
	})

	Context("Service", func() {
		It("je ne sais pas ce que je fais", func() {
			var ctx = context.TODO()
			services, err := broker.Services(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			txt, err := json.Marshal(services[0])
			fmt.Println("test:", string(txt))
		})
	})

})
