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
	BeforeAll(func(ctx SpecContext) {
		// Arrange
		output, err := fixture.run(ctx, "make", "-f", "terraform/Makefile", "ROOT=gitops", "apply")
		Expect(err).NotTo(HaveOccurred(), output)
		output, err = fixture.kubectl(ctx, "apply", "--server-side",
			"-f", "clusters/dev/gitops/projects.yaml",
			"-f", "clusters/dev/gitops/external-secrets.yaml",
			"-f", "clusters/dev/gitops/reloader.yaml")
		Expect(err).NotTo(HaveOccurred(), output)

		for _, application := range []string{"monadiam-external-secrets", "monadiam-reloader"} {
			output, err = fixture.kubectl(ctx, "wait", "application/"+application, "-n", "argocd",
				"--for=jsonpath={.status.health.status}=Healthy", "--timeout=300s")
			Expect(err).NotTo(HaveOccurred(), output)
		}

		output, err = fixture.kubectl(ctx, "apply", "--server-side", "-k", "infrastructure/security/application-secrets")
		Expect(err).NotTo(HaveOccurred(), output)
		output, err = fixture.kubectl(ctx, "wait", "secretstore/kubernetes-data", "-n", "monadiam-apps",
			"--for=condition=Ready", "--timeout=60s")
		Expect(err).NotTo(HaveOccurred(), output)
	}, NodeTimeout(27*time.Minute))

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
				output, err := fixture.kubectl(ctx, "get", "secret/application-secret-probe",
					"-n", "monadiam-apps", "-o", "json")

				// Assert
				Expect(err).NotTo(HaveOccurred(), output)
				Expect(json.Unmarshal([]byte(output), &destination)).To(Succeed())
				Expect(destination.Data).To(Equal(map[string][]byte{"selected": []byte("initial")}))
			}, SpecTimeout(time.Minute))
		})
	})

	Describe("[Service] - Reloader", func() {
		Describe("[Behavior] - secret rotation", func() {
			It("[case] - delivers a rotated value to a new container without CLI reconciliation", func(ctx SpecContext) {
				// Arrange
				manifestPath, err := filepath.Abs("testdata/application-secrets/update.yaml")
				Expect(err).NotTo(HaveOccurred())
				output, err := fixture.kubectl(ctx, "exec", "deployment/application-secret-probe",
					"-n", "monadiam-apps", "--", "printenv", "PROBE_VALUE")
				Expect(err).NotTo(HaveOccurred(), output)
				Expect(strings.TrimSpace(output)).To(Equal("initial"))

				// Act
				output, err = fixture.kubectl(ctx, "apply", "-f", manifestPath)

				// Assert
				Expect(err).NotTo(HaveOccurred(), output)
				Eventually(func() (string, error) {
					output, err := fixture.kubectl(ctx, "exec", "deployment/application-secret-probe",
						"-n", "monadiam-apps", "--", "printenv", "PROBE_VALUE")

					return strings.TrimSpace(output), err
				}).WithContext(ctx).WithTimeout(2 * time.Minute).WithPolling(2 * time.Second).Should(Equal("rotated"))
				output, err = fixture.kubectl(ctx, "rollout", "status", "deployment/application-secret-probe",
					"-n", "monadiam-apps", "--timeout=60s")
				Expect(err).NotTo(HaveOccurred(), output)
			}, SpecTimeout(4*time.Minute))
		})
	})
})
