//go:build integration

package integration

import (
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stand infrastructure integration")
}

var fixture clusterFixture

var _ = BeforeSuite(func(ctx SpecContext) {
	configuration, _ := GinkgoConfiguration()
	Expect(configuration.ParallelTotal).To(Equal(1), "integration tests share one stand; run without -p")
	action := os.Getenv("INTEGRATION_ACTION")
	Expect(action).To(BeElementOf("prepare", "check", "cleanup"), "use make itest-prepare, itest, itest-slow or itest-cleanup")
	By("Opening the saved integration environment: " + action)
	fixture.open(action)
	if action == "check" {
		By("Verifying the identity of the prepared test cluster")
		fixture.attach(ctx)
		fixture.prepare(ctx)
	}
}, NodeTimeout(2*time.Minute))

var _ = AfterEach(func(ctx SpecContext) {
	if CurrentSpecReport().Failed() {
		By("Collecting diagnostics after failure")
		fixture.diagnostics(ctx)
	}
}, NodeTimeout(2*time.Minute))
