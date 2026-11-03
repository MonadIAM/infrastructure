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
			output, err := fixture.compose(ctx, "down", "--volumes")
			Expect(err).NotTo(HaveOccurred(), output)
		}, NodeTimeout(time.Minute))

		output, err := fixture.compose(ctx, "up", "-d", "--no-deps", "--wait", "vault")
		Expect(err).NotTo(HaveOccurred(), output)
		Expect(fixture.initialize(ctx)).To(Succeed())
		before, err := os.ReadFile(filepath.Join(fixture.directory, "init", "root-token"))
		Expect(err).NotTo(HaveOccurred())

		// Act
		output, err = fixture.compose(ctx, "stop", "vault")
		Expect(err).NotTo(HaveOccurred(), output)
		output, err = fixture.compose(ctx, "up", "-d", "--no-deps", "--wait", "vault")
		Expect(err).NotTo(HaveOccurred(), output)
		Expect(fixture.initialize(ctx)).To(Succeed())

		// Assert
		after, err := os.ReadFile(filepath.Join(fixture.directory, "init", "root-token"))
		Expect(err).NotTo(HaveOccurred())
		Expect(after).To(Equal(before))
		output, err = fixture.compose(ctx, "ps", "--status", "running", "--services")
		Expect(err).NotTo(HaveOccurred(), output)
		Expect(output).To(Equal("vault\n"))
	}, SpecTimeout(5*time.Minute))
})
