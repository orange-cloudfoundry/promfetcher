package models_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/orange-cloudfoundry/promfetcher/models"
)

var _ = Describe("Route", func() {

	var routes models.Routes

	BeforeEach(func() {
		routes = make(models.Routes)

		r1 := make([]*models.Route, 0)
		r1 = append(r1, &models.Route{
			Address: "test1.cf.internal",
			Tags: models.Tags{
				ProcessType:      "web",
				OrganizationName: "myorg1",
				SpaceName:        "myspace1",
				AppName:          "test1",
				AppID:            "a758f25d-2d01-419e-b63b-de3aabcd9e15",
			},
		})
		routes["route1"] = r1

		r2 := make([]*models.Route, 0)
		r2 = append(r2, &models.Route{
			Address: "test2.cf.internal",
			Tags: models.Tags{
				ProcessType:      "web",
				OrganizationName: "myorg1",
				SpaceName:        "myspace2",
				AppName:          "test2",
				AppID:            "b758f25d-2d01-419e-b63b-de3aabcd9e15",
			},
		})
		routes["route2"] = r2

		r3 := make([]*models.Route, 0)
		r3 = append(r3, &models.Route{
			Address: "test3.cf.internal",
			Tags: models.Tags{
				ProcessType:      "web",
				OrganizationName: "myorg2",
				SpaceName:        "myspace1",
				AppName:          "test3",
				AppID:            "c758f25d-2d01-419e-b63b-de3aabcd9e15",
			},
		})
		routes["route3"] = r3
	})

	Context("Search routes", func() {
		It("finds route by name", func() {
			rts := routes.FindByRouteName("route1")
			fmt.Printf("%s\n", rts[0].Address)
			Expect(len(rts)).To(Equal(1))
		})
		It("finds route by org/space/app", func() {
			rts := routes.FindByOrgSpaceName("myorg1", "myspace2", "test2")
			Expect(len(rts)).To(Equal(1))
		})
		It("finds route by app id", func() {
			rts := routes.FindById("a758f25d-2d01-419e-b63b-de3aabcd9e15")
			Expect(len(rts)).To(Equal(1))
		})
		It("finds route by app id with Find function", func() {
			rts := routes.Find("c758f25d-2d01-419e-b63b-de3aabcd9e15")
			Expect(len(rts)).To(Equal(1))
		})
		It("finds route by org/space/app with Find function", func() {
			rts := routes.Find("myorg2/myspace1/test3")
			Expect(len(rts)).To(Equal(1))
		})
		It("finds route by name with Find function", func() {
			rts := routes.Find("route2")
			Expect(len(rts)).To(Equal(1))
		})
		It("does not find unknown route", func() {
			rts := routes.Find("unknown")
			Expect(len(rts)).To(Equal(0))
		})
		It("does not find route with bad org/space/app", func() {
			rts := routes.FindByOrgSpaceName("myorg2", "myspace2", "test2")
			Expect(len(rts)).To(Equal(0))
		})
	})
})
