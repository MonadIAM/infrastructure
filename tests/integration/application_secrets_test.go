//go:build integration

package integration

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[Infrastructure] - Application secrets", Ordered, Label("integration", "application-secrets"), func() {
	BeforeEach(func(ctx SpecContext) {
		// Arrange
		manifestPath, err := filepath.Abs("testdata/application-secrets/probe.yaml")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func(ctx SpecContext) {
			// Cleanup
			output, err := fixture.kubectl(ctx, "delete", "-f", manifestPath, "--ignore-not-found", "--timeout=60s")
			Expect(err).NotTo(HaveOccurred(), output)
			output, err = fixture.kubectl(ctx, "delete", "secret/application-secret-probe",
				"-n", "monadiam-apps", "--ignore-not-found")
			Expect(err).NotTo(HaveOccurred(), output)
		}, NodeTimeout(2*time.Minute))
		output, err := fixture.kubectl(ctx, "apply", "-f", manifestPath)
		Expect(err).NotTo(HaveOccurred(), output)
		output, err = fixture.kubectl(ctx, "wait", "externalsecret/application-secret-probe",
			"-n", "monadiam-apps", "--for=condition=Ready", "--timeout=60s")
		Expect(err).NotTo(HaveOccurred(), output)
		output, err = fixture.kubectl(ctx, "rollout", "status", "deployment/application-secret-probe",
			"-n", "monadiam-apps", "--timeout=120s")
		Expect(err).NotTo(HaveOccurred(), output)
	}, NodeTimeout(4*time.Minute))

	Describe("[Service] - External Secrets Operator", func() {
		Describe("[Behavior] - selected-key delivery", func() {
			It("[case] - copies only the selected key across namespaces", func(ctx SpecContext) {
				// Arrange
				var destination resource

				// Act
				output, err := fixture.kubectlSensitive(ctx, "get", "secret/application-secret-probe",
					"-n", "monadiam-apps", "-o", "json")

				// Assert
				Expect(err).NotTo(HaveOccurred(), output)
				Expect(json.Unmarshal([]byte(output), &destination)).To(Succeed())
				Expect(destination.Data).To(Equal(map[string][]byte{"selected": []byte("initial")}))
			}, NodeTimeout(time.Minute))
		})
	})

	Describe("[Service] - Reloader", func() {
		Describe("[Behavior] - secret rotation", func() {
			It("[case] - delivers a rotated value to a new container without CLI reconciliation", func(ctx SpecContext) {
				// Arrange
				manifestPath, err := filepath.Abs("testdata/application-secrets/update.yaml")
				Expect(err).NotTo(HaveOccurred())
				output, err := fixture.kubectlSensitive(ctx, "exec", "deployment/application-secret-probe",
					"-n", "monadiam-apps", "--", "printenv", "PROBE_VALUE")
				Expect(err).NotTo(HaveOccurred(), output)
				Expect(strings.TrimSpace(output)).To(Equal("initial"))

				// Act
				output, err = fixture.kubectl(ctx, "apply", "-f", manifestPath)

				// Assert
				Expect(err).NotTo(HaveOccurred(), output)
				Eventually(func() (string, error) {
					output, err := fixture.kubectlSensitive(ctx, "exec", "deployment/application-secret-probe",
						"-n", "monadiam-apps", "--", "printenv", "PROBE_VALUE")

					return strings.TrimSpace(output), err
				}).WithContext(ctx).WithTimeout(2 * time.Minute).WithPolling(2 * time.Second).Should(Equal("rotated"))
				output, err = fixture.kubectl(ctx, "rollout", "status", "deployment/application-secret-probe",
					"-n", "monadiam-apps", "--timeout=60s")
				Expect(err).NotTo(HaveOccurred(), output)
			}, NodeTimeout(4*time.Minute))
		})
	})
})
