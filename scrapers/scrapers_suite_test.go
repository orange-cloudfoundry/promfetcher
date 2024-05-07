package scrapers_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestScrapers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scrapers Suite")
}
