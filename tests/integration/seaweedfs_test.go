//go:build integration

package integration

import (
	"time"

	"github.com/minio/minio-go/v7"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[Infrastructure] - SeaweedFS", Label("integration", "seaweedfs"), func() {
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

})
