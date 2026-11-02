//go:build integration

package integration

import (
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[Infrastructure] - Shared schemas", Ordered, Label("integration", "schemas"), func() {
	var publicationPatch string
	var previousJobUID string

	BeforeAll(func(ctx SpecContext) {
		// Arrange
		manifestPath, err := filepath.Abs("testdata/shared-schemas/application.yaml")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func(ctx SpecContext) {
			// Cleanup
			output, err := fixture.kubectl(ctx, "delete", "-f", manifestPath, "--ignore-not-found")
			Expect(err).NotTo(HaveOccurred(), output)
		}, NodeTimeout(time.Minute))
		output, err := fixture.kubectl(ctx, "apply", "-f", manifestPath)
		Expect(err).NotTo(HaveOccurred(), output)
	}, NodeTimeout(time.Minute))

	BeforeEach(func(ctx SpecContext) {
		// Arrange
		publicationPatch = fixture.schemaSyncPatch(ctx)
		var err error
		previousJobUID, err = fixture.kubectl(ctx, "get", "job/shared-schemas", "-n", "monadiam-messaging",
			"--ignore-not-found", "-o", "jsonpath={.metadata.uid}")
		Expect(err).NotTo(HaveOccurred(), previousJobUID)
	}, NodeTimeout(time.Minute))

	Describe("[Service] - Argo CD", func() {
		Describe("[Behavior] - catalog publication", func() {
			It("[case] - completes the Sync hook and publishes the catalog", func(ctx SpecContext) {
				// Arrange
				application := "application/shared-schemas-probe"

				// Act
				output, err := fixture.kubectl(ctx, "patch", application, "-n", "argocd",
					"--type=merge", "--patch-file", publicationPatch)

				// Assert
				Expect(err).NotTo(HaveOccurred(), output)
				Eventually(func() (string, error) {
					return fixture.kubectl(ctx, "get", "job/shared-schemas", "-n", "monadiam-messaging",
						"--ignore-not-found", "-o", "jsonpath={.metadata.uid}")
				}).WithContext(ctx).WithTimeout(time.Minute).Should(And(Not(BeEmpty()), Not(Equal(previousJobUID))))
				output, err = fixture.kubectl(ctx, "wait", "job/shared-schemas", "-n", "monadiam-messaging",
					"--for=condition=Complete", "--timeout=300s")
				Expect(err).NotTo(HaveOccurred(), output)
				output, err = fixture.kubectl(ctx, "wait", application, "-n", "argocd",
					"--for=jsonpath={.status.operationState.phase}=Succeeded", "--timeout=180s")
				Expect(err).NotTo(HaveOccurred(), output)
				output, err = fixture.kubectl(ctx, "logs", "job/shared-schemas", "-n", "monadiam-messaging")
				Expect(err).NotTo(HaveOccurred(), output)
				Expect(output).To(ContainSubstring("[SchemaRegistry]: applied - monadiam.grpc/access-control"))
				Expect(output).To(ContainSubstring("[SchemaRegistry]: applied - account-value"))
				Expect(output).To(ContainSubstring("[SchemaRegistry]: applied - realm-value"))
			}, NodeTimeout(10*time.Minute))
		})

		Describe("[Behavior] - repeated publication", func() {
			It("[case] - recreates the completed Job on an explicit sync", func(ctx SpecContext) {
				// Arrange
				application := "application/shared-schemas-probe"
				Expect(previousJobUID).NotTo(BeEmpty())

				// Act
				output, err := fixture.kubectl(ctx, "patch", application, "-n", "argocd",
					"--type=merge", "--patch-file", publicationPatch)

				// Assert
				Expect(err).NotTo(HaveOccurred(), output)
				Eventually(func() (string, error) {
					return fixture.kubectl(ctx, "get", "job/shared-schemas", "-n", "monadiam-messaging",
						"--ignore-not-found", "-o", "jsonpath={.metadata.uid}")
				}).WithContext(ctx).WithTimeout(time.Minute).Should(And(Not(BeEmpty()), Not(Equal(previousJobUID))))
				output, err = fixture.kubectl(ctx, "wait", "job/shared-schemas", "-n", "monadiam-messaging",
					"--for=condition=Complete", "--timeout=300s")
				Expect(err).NotTo(HaveOccurred(), output)
				output, err = fixture.kubectl(ctx, "wait", application, "-n", "argocd",
					"--for=jsonpath={.status.operationState.phase}=Succeeded", "--timeout=180s")
				Expect(err).NotTo(HaveOccurred(), output)
			}, NodeTimeout(10*time.Minute))
		})
	})
})
