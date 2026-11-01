//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[Infrastructure] - Vault server", Label("integration", "vault-server"), func() {
	It("[case] - installs a fresh server and preserves initialization after restart", func(ctx SpecContext) {
		// Arrange
		fixture := newVaultServerFixture()
		DeferCleanup(func(ctx SpecContext) {
			for _, arguments := range [][]string{
				{"rm", "-f", fixture.name},
				{"volume", "rm", fixture.name + "_vault_data", fixture.name + "_vault_init"},
				{"network", "rm", fixture.name},
			} {
				output, err := fixture.docker(ctx, arguments...)
				Expect(err).NotTo(HaveOccurred(), output)
			}
		}, NodeTimeout(time.Minute))

		output, err := fixture.terraform(ctx, "apply")
		Expect(err).NotTo(HaveOccurred(), output)
		Expect(fixture.initialize(ctx)).To(Succeed())
		before, err := os.ReadFile(filepath.Join(fixture.directory, "init", "root-token"))
		Expect(err).NotTo(HaveOccurred())

		// Act
		output, err = fixture.docker(ctx, "stop", fixture.name)
		Expect(err).NotTo(HaveOccurred(), output)
		output, err = fixture.terraform(ctx, "apply")
		Expect(err).NotTo(HaveOccurred(), output)
		Expect(fixture.initialize(ctx)).To(Succeed())

		// Assert
		after, err := os.ReadFile(filepath.Join(fixture.directory, "init", "root-token"))
		Expect(err).NotTo(HaveOccurred())
		Expect(after).To(Equal(before))
		output, err = fixture.terraform(ctx, "plan")
		Expect(err).NotTo(HaveOccurred(), output)
	}, SpecTimeout(5*time.Minute))
})
