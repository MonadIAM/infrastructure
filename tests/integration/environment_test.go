//go:build integration

package integration

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[Infrastructure] - Test environment", Label("environment"), func() {
	Describe("[Behavior] - explicit lifecycle", func() {
		It("[case] - prepares a persistent isolated stand", Label("environment-prepare"), func(ctx SpecContext) {
			// Arrange
			Expect(os.Getenv("INTEGRATION_ACTION")).To(Equal("prepare"))
			fixture.prepare(ctx)

			// Act
			By("Deploying the platform through infra cluster start")
			output, err := fixture.cli(ctx, "cluster", "start")
			Expect(err).NotTo(HaveOccurred(), output)
			fixture.prepareGitOps(ctx)

			// Assert
			fixture.recordCluster(ctx)
			fixture.attach(ctx)
		}, NodeTimeout(60*time.Minute))

		It("[case] - deletes only the saved test stand", Label("environment-cleanup"), func(ctx SpecContext) {
			// Arrange
			Expect(os.Getenv("INTEGRATION_ACTION")).To(Equal("cleanup"))

			// Act
			fixture.cleanup(ctx)

			// Assert
			Expect(os.RemoveAll(fixture.directory)).To(Succeed())
		}, NodeTimeout(5*time.Minute))
	})
})
