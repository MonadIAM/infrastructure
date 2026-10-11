//go:build integration

package integration

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[Infrastructure] - Stand bootstrap", Ordered, Label("integration"), func() {
	var fixture clusterFixture

	BeforeAll(func(ctx SpecContext) {
		fixture = clusterFixture{
			name:      fmt.Sprintf("infra-test-%d", time.Now().UnixNano()),
			directory: GinkgoT().TempDir(),
		}
		DeferCleanup(func(ctx SpecContext) {
			fixture.cleanup(ctx)
		}, NodeTimeout(5*time.Minute))

		fixture.prepare(ctx)
		output, err := fixture.cli(ctx, "cluster", "start")
		Expect(err).NotTo(HaveOccurred(), output)
	}, NodeTimeout(35*time.Minute))

	AfterEach(func(ctx SpecContext) {
		if CurrentSpecReport().Failed() {
			fixture.diagnostics(ctx)
		}
	}, NodeTimeout(2*time.Minute))

	Describe("[Behavior] - initial deployment", func() {
		It("[case] - initializes Vault, synchronizes credentials and accepts PostgreSQL writes", func(ctx SpecContext) {
			// Arrange

			// Act
			vault, vaultErr := fixture.vault(ctx)
			state, stateErr := fixture.state(ctx)
			seedErr := fixture.seed(ctx)
			value, readErr := fixture.postgresql(ctx, "SELECT value FROM integration_probe WHERE id = 1")

			// Assert
			Expect(vaultErr).NotTo(HaveOccurred())
			Expect(vault.Initialized).To(BeTrue())
			Expect(vault.Sealed).To(BeFalse())
			Expect(vault.FileMode).To(BeNumerically("==", 0600))
			Expect(vault.DirectoryMode).To(BeNumerically("==", 0700))
			Expect(stateErr).NotTo(HaveOccurred())
			Expect(state.Namespace).NotTo(BeEmpty())
			Expect(state.Volumes).To(HaveKey("vault/data-vault-0"))
			Expect(state.Volumes).To(HaveKey("monadiam-data/postgresql-main-1"))
			Expect(state.Credentials).To(HaveLen(12))
			Expect(seedErr).NotTo(HaveOccurred())
			Expect(readErr).NotTo(HaveOccurred(), value)
			Expect(value).To(Equal(fixture.name))
		}, SpecTimeout(35*time.Minute))
	})

	Describe("[Behavior] - repeated reconciliation", func() {
		It("[case] - preserves volumes, Vault material, credentials and PostgreSQL rows", func(ctx SpecContext) {
			// Arrange
			original, err := fixture.state(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(fixture.seed(ctx)).To(Succeed())

			// Act
			output, reconcileErr := fixture.cli(ctx, "platform", "reconcile")
			state, stateErr := fixture.state(ctx)
			value, readErr := fixture.postgresql(ctx, "SELECT value FROM integration_probe WHERE id = 1")

			// Assert
			Expect(reconcileErr).NotTo(HaveOccurred(), output)
			Expect(stateErr).NotTo(HaveOccurred())
			Expect(state).To(Equal(original))
			Expect(readErr).NotTo(HaveOccurred(), value)
			Expect(value).To(Equal(fixture.name))
		}, SpecTimeout(35*time.Minute))
	})

	Describe("[Behavior] - cluster deletion protection", func() {
		It("[case] - refuses recreation without confirmation and preserves the running cluster", func(ctx SpecContext) {
			// Arrange
			original, err := fixture.state(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(fixture.seed(ctx)).To(Succeed())

			// Act
			output, recreateErr := fixture.cli(ctx, "cluster", "recreate")
			state, stateErr := fixture.state(ctx)
			value, readErr := fixture.postgresql(ctx, "SELECT value FROM integration_probe WHERE id = 1")

			// Assert
			Expect(recreateErr).To(HaveOccurred())
			Expect(output).To(ContainSubstring("pass --confirm-delete"))
			Expect(stateErr).NotTo(HaveOccurred())
			Expect(state).To(Equal(original))
			Expect(readErr).NotTo(HaveOccurred(), value)
			Expect(value).To(Equal(fixture.name))
		}, SpecTimeout(3*time.Minute))
	})

	Describe("[Behavior] - cluster restart", func() {
		It("[case] - unseals Vault and restores access to existing PostgreSQL data", func(ctx SpecContext) {
			// Arrange
			original, err := fixture.state(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(fixture.seed(ctx)).To(Succeed())

			// Act
			output, restartErr := fixture.cli(ctx, "cluster", "restart")
			state, stateErr := fixture.state(ctx)
			vault, vaultErr := fixture.vault(ctx)
			value, readErr := fixture.postgresql(ctx, "SELECT value FROM integration_probe WHERE id = 1")

			// Assert
			Expect(restartErr).NotTo(HaveOccurred(), output)
			Expect(stateErr).NotTo(HaveOccurred())
			Expect(state).To(Equal(original))
			Expect(vaultErr).NotTo(HaveOccurred())
			Expect(vault.Initialized).To(BeTrue())
			Expect(vault.Sealed).To(BeFalse())
			Expect(readErr).NotTo(HaveOccurred(), value)
			Expect(value).To(Equal(fixture.name))
		}, SpecTimeout(35*time.Minute))
	})
})
