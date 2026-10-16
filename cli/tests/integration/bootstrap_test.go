//go:build integration

package integration

import (
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
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
			values, readErr := fixture.postgresqlValues(ctx)

			// Assert
			Expect(vaultErr).NotTo(HaveOccurred())
			Expect(vault.Initialized).To(BeTrue())
			Expect(vault.Sealed).To(BeFalse())
			Expect(vault.FileMode).To(BeNumerically("==", 0600))
			Expect(vault.DirectoryMode).To(BeNumerically("==", 0700))
			Expect(stateErr).NotTo(HaveOccurred())
			Expect(state.Namespace).NotTo(BeEmpty())
			Expect(state.Volumes).To(HaveKey("vault/data-vault-0"))
			for _, service := range postgresqlServices {
				for instance := 1; instance <= 2; instance++ {
					claim := fmt.Sprintf("monadiam-data/%s-postgresql-%d", service, instance)
					Expect(state.Volumes).To(HaveKey(claim))
					Expect(state.Volumes).To(HaveKey(claim + "-wal"))
				}
			}

			Expect(state.Volumes).NotTo(HaveKey("monadiam-data/postgresql-main-1"))
			Expect(state.Volumes).To(HaveKey("monadiam-data/data-monadiam-data-seaweedfs-main-master-0"))
			Expect(state.Volumes).To(HaveKey("monadiam-data/data-seaweedfs-main-volume-0"))
			Expect(state.Volumes).To(HaveKey("monadiam-data/data-filer-seaweedfs-main-filer-0"))
			Expect(state.Credentials).To(HaveLen(29))
			Expect(seedErr).NotTo(HaveOccurred())
			Expect(readErr).NotTo(HaveOccurred())
			Expect(values).To(HaveLen(6))
			Expect(values).To(HaveEach(Equal(fixture.name)))
		}, SpecTimeout(35*time.Minute))
	})

	Describe("[Behavior] - archive storage", func() {
		It("[case] - preserves archiver writes and enforces versioning, retention and deletion restrictions", func(ctx SpecContext) {
			// Arrange
			settings := fixture.storage(ctx)
			admin, err := newStorageClient(settings, true)
			Expect(err).NotTo(HaveOccurred())
			archiver, err := newStorageClient(settings, false)
			Expect(err).NotTo(HaveOccurred())
			versions, err := archiver.seed(ctx, fixture.name)
			Expect(err).NotTo(HaveOccurred())

			// Act
			readErr := archiver.read(ctx, versions, fixture.name)
			var retentionErrors []error
			var deleteErrors []error

			for _, bucket := range settings.Buckets {
				retentionErrors = append(retentionErrors, admin.retention(ctx, bucket))
				version := versions[bucket.Name]
				deleteErrors = append(deleteErrors,
					archiver.client.RemoveObject(ctx, bucket.Name, "integration/probe.json", minio.RemoveObjectOptions{}),
					archiver.client.RemoveObject(ctx, bucket.Name, "integration/probe.json", minio.RemoveObjectOptions{VersionID: version}),
					admin.client.RemoveObject(ctx, bucket.Name, "integration/probe.json", minio.RemoveObjectOptions{VersionID: version}),
				)
			}

			createErr := archiver.client.MakeBucket(ctx, "unauthorized-archive", minio.MakeBucketOptions{Region: settings.Region})
			preservedErr := archiver.read(ctx, versions, fixture.name)

			// Assert
			Expect(readErr).NotTo(HaveOccurred())
			Expect(retentionErrors).To(HaveEach(Succeed()))
			Expect(deleteErrors).To(HaveEach(HaveOccurred()))
			Expect(createErr).To(HaveOccurred())
			Expect(preservedErr).NotTo(HaveOccurred())
		}, SpecTimeout(3*time.Minute))
	})

	Describe("[Behavior] - repeated reconciliation", func() {
		It("[case] - preserves volumes, credentials, PostgreSQL rows and archive objects", func(ctx SpecContext) {
			// Arrange
			original, err := fixture.state(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(fixture.seed(ctx)).To(Succeed())
			storage, err := newStorageClient(fixture.storage(ctx), false)
			Expect(err).NotTo(HaveOccurred())
			versions, err := storage.seed(ctx, fixture.name)
			Expect(err).NotTo(HaveOccurred())

			// Act
			output, reconcileErr := fixture.cli(ctx, "platform", "reconcile")
			state, stateErr := fixture.state(ctx)
			values, readErr := fixture.postgresqlValues(ctx)
			storage, storageErr := newStorageClient(fixture.storage(ctx), false)
			var archiveErr error
			if storageErr == nil {
				archiveErr = storage.read(ctx, versions, fixture.name)
			}

			// Assert
			Expect(reconcileErr).NotTo(HaveOccurred(), output)
			Expect(stateErr).NotTo(HaveOccurred())
			Expect(state).To(Equal(original))
			Expect(readErr).NotTo(HaveOccurred())
			Expect(values).To(HaveLen(6))
			Expect(values).To(HaveEach(Equal(fixture.name)))
			Expect(storageErr).NotTo(HaveOccurred())
			Expect(archiveErr).NotTo(HaveOccurred())
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
			values, readErr := fixture.postgresqlValues(ctx)

			// Assert
			Expect(recreateErr).To(HaveOccurred())
			Expect(output).To(ContainSubstring("pass --confirm-delete"))
			Expect(stateErr).NotTo(HaveOccurred())
			Expect(state).To(Equal(original))
			Expect(readErr).NotTo(HaveOccurred())
			Expect(values).To(HaveLen(6))
			Expect(values).To(HaveEach(Equal(fixture.name)))
		}, SpecTimeout(3*time.Minute))
	})

	Describe("[Behavior] - cluster restart", func() {
		It("[case] - unseals Vault and restores access to PostgreSQL data and archive objects", func(ctx SpecContext) {
			// Arrange
			original, err := fixture.state(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(fixture.seed(ctx)).To(Succeed())
			storage, err := newStorageClient(fixture.storage(ctx), false)
			Expect(err).NotTo(HaveOccurred())
			versions, err := storage.seed(ctx, fixture.name)
			Expect(err).NotTo(HaveOccurred())

			// Act
			output, restartErr := fixture.cli(ctx, "cluster", "restart")
			state, stateErr := fixture.state(ctx)
			vault, vaultErr := fixture.vault(ctx)
			values, readErr := fixture.postgresqlValues(ctx)
			storage, storageErr := newStorageClient(fixture.storage(ctx), false)
			var archiveErr error
			if storageErr == nil {
				archiveErr = storage.read(ctx, versions, fixture.name)
			}

			// Assert
			Expect(restartErr).NotTo(HaveOccurred(), output)
			Expect(stateErr).NotTo(HaveOccurred())
			Expect(state).To(Equal(original))
			Expect(vaultErr).NotTo(HaveOccurred())
			Expect(vault.Initialized).To(BeTrue())
			Expect(vault.Sealed).To(BeFalse())
			Expect(storageErr).NotTo(HaveOccurred())
			Expect(archiveErr).NotTo(HaveOccurred())
			Expect(readErr).NotTo(HaveOccurred())
			Expect(values).To(HaveLen(6))
			Expect(values).To(HaveEach(Equal(fixture.name)))
		}, SpecTimeout(35*time.Minute))
	})
})
