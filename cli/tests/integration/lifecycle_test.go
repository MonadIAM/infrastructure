//go:build integration

package integration

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[Infrastructure] - Stand lifecycle", Label("integration", "lifecycle"), func() {
	Describe("[Behavior] - repeated reconciliation", func() {
		It("[case] - preserves volumes, credentials, PostgreSQL rows, Redis values and archive objects", func(ctx SpecContext) {
			// Arrange
			original, err := fixture.state(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(fixture.seed(ctx)).To(Succeed())
			Expect(fixture.seedRedis(ctx)).To(Succeed())
			storage, err := newStorageClient(fixture.storage(ctx), false)
			Expect(err).NotTo(HaveOccurred())
			versions, err := storage.seed(ctx, fixture.name)
			Expect(err).NotTo(HaveOccurred())

			// Act
			output, reconcileErr := fixture.cli(ctx, "platform", "reconcile")
			state, stateErr := fixture.state(ctx)
			values, readErr := fixture.postgresqlValues(ctx)
			redisValues, redisErr := fixture.redisValues(ctx)
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
			Expect(values).To(HaveLen(len(services)))
			Expect(values).To(HaveEach(Equal(fixture.name)))
			Expect(redisErr).NotTo(HaveOccurred())
			Expect(redisValues).To(HaveLen(len(services)))
			Expect(redisValues).To(HaveEach(Equal(fixture.name)))
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
			Expect(fixture.seedRedis(ctx)).To(Succeed())

			// Act
			output, recreateErr := fixture.cli(ctx, "cluster", "recreate")
			state, stateErr := fixture.state(ctx)
			values, readErr := fixture.postgresqlValues(ctx)
			redisValues, redisErr := fixture.redisValues(ctx)

			// Assert
			Expect(recreateErr).To(HaveOccurred())
			Expect(output).To(ContainSubstring("pass --confirm-delete"))
			Expect(stateErr).NotTo(HaveOccurred())
			Expect(state).To(Equal(original))
			Expect(readErr).NotTo(HaveOccurred())
			Expect(values).To(HaveLen(len(services)))
			Expect(values).To(HaveEach(Equal(fixture.name)))
			Expect(redisErr).NotTo(HaveOccurred())
			Expect(redisValues).To(HaveLen(len(services)))
			Expect(redisValues).To(HaveEach(Equal(fixture.name)))
		}, SpecTimeout(3*time.Minute))
	})

	Describe("[Behavior] - cluster restart", func() {
		It("[case] - unseals Vault and restores access to PostgreSQL, Redis and archive objects", func(ctx SpecContext) {
			// Arrange
			original, err := fixture.state(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(fixture.seed(ctx)).To(Succeed())
			Expect(fixture.seedRedis(ctx)).To(Succeed())
			storage, err := newStorageClient(fixture.storage(ctx), false)
			Expect(err).NotTo(HaveOccurred())
			versions, err := storage.seed(ctx, fixture.name)
			Expect(err).NotTo(HaveOccurred())

			// Act
			output, restartErr := fixture.cli(ctx, "cluster", "restart")
			state, stateErr := fixture.state(ctx)
			vault, vaultErr := fixture.vault(ctx)
			values, readErr := fixture.postgresqlValues(ctx)
			redisValues, redisErr := fixture.redisValues(ctx)
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
			Expect(values).To(HaveLen(len(services)))
			Expect(values).To(HaveEach(Equal(fixture.name)))
			Expect(redisErr).NotTo(HaveOccurred())
			Expect(redisValues).To(HaveLen(len(services)))
			Expect(redisValues).To(HaveEach(Equal(fixture.name)))
		}, SpecTimeout(35*time.Minute))
	})
})
