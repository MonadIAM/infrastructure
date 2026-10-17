//go:build integration

package integration

import (
	"fmt"
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
	fixture = clusterFixture{name: fmt.Sprintf("infra-test-%d", time.Now().UnixNano()), directory: GinkgoT().TempDir()}
	DeferCleanup(func(ctx SpecContext) { fixture.cleanup(ctx) }, NodeTimeout(5*time.Minute))
	fixture.prepare(ctx)
	output, err := fixture.cli(ctx, "cluster", "start")
	if err != nil {
		fixture.diagnostics(ctx)
	}

	Expect(err).NotTo(HaveOccurred(), output)
}, NodeTimeout(35*time.Minute))

var _ = AfterEach(func(ctx SpecContext) {
	if CurrentSpecReport().Failed() {
		fixture.diagnostics(ctx)
	}
}, NodeTimeout(2*time.Minute))
