//go:build integration

package integration

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[Infrastructure] - Stand bootstrap", Label("integration", "bootstrap"), func() {
	Describe("[Behavior] - initial deployment", func() {
		It("[case] - initializes Vault and synchronizes credentials", func(ctx SpecContext) {
			// Arrange

			// Act
			vault, vaultErr := fixture.vault(ctx)
			state, stateErr := fixture.state(ctx)

			// Assert
			Expect(vaultErr).NotTo(HaveOccurred())
			Expect(vault.Initialized).To(BeTrue())
			Expect(vault.Sealed).To(BeFalse())
			Expect(vault.FileMode).To(BeNumerically("==", 0600))
			Expect(vault.DirectoryMode).To(BeNumerically("==", 0700))
			Expect(stateErr).NotTo(HaveOccurred())
			Expect(state.Namespace).NotTo(BeEmpty())
			Expect(state.Volumes).To(HaveKey("vault/data-vault-0"))
			Expect(state.Volumes).To(HaveKey("monadiam-data/data-monadiam-data-seaweedfs-main-master-0"))
			Expect(state.Volumes).To(HaveKey("monadiam-data/data-seaweedfs-main-volume-0"))
			Expect(state.Volumes).To(HaveKey("monadiam-data/data-filer-seaweedfs-main-filer-0"))
			Expect(state.Credentials).To(HaveLen(26))
		}, SpecTimeout(35*time.Minute))
	})
})
