//go:build integration

package integration

import (
	"encoding/json"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[Infrastructure] - GitOps", Ordered, Label("integration", "gitops"), func() {
	BeforeAll(func(ctx SpecContext) {
		output, err := fixture.run(ctx, "make", "-f", "terraform/Makefile", "ROOT=gitops", "apply")
		Expect(err).NotTo(HaveOccurred(), output)
	}, NodeTimeout(15*time.Minute))

	Describe("[Behavior] - controller deployment", func() {
		DescribeTable("[case] - starts the workload", func(ctx SpecContext, workload string) {
			// Arrange
			namespace := "argocd"

			// Act
			output, err := fixture.kubectl(ctx, "rollout", "status", workload, "-n", namespace, "--timeout=120s")

			// Assert
			Expect(err).NotTo(HaveOccurred(), output)
		},
			Entry("application controller", "statefulset/argocd-application-controller", SpecTimeout(3*time.Minute)),
			Entry("repository server", "deployment/argocd-repo-server", SpecTimeout(3*time.Minute)),
			Entry("API server", "deployment/argocd-server", SpecTimeout(3*time.Minute)),
			Entry("image updater", "deployment/argocd-image-updater-controller", SpecTimeout(3*time.Minute)),
		)
	})

	Describe("[Behavior] - Terraform reconciliation", func() {
		It("[case] - has no drift after installation", func(ctx SpecContext) {
			// Arrange
			arguments := []string{"TF_CLI_ARGS_plan=-detailed-exitcode", "make", "-f", "terraform/Makefile", "ROOT=gitops", "plan"}

			// Act
			output, err := fixture.run(ctx, "env", arguments...)

			// Assert
			Expect(err).NotTo(HaveOccurred(), output)
		}, SpecTimeout(5*time.Minute))
	})

	Describe("[Behavior] - repository access", func() {
		It("[case] - retrieves manifests without syncing workloads", func(ctx SpecContext) {
			// Arrange
			manifestPath, err := filepath.Abs("testdata/gitops/repository-probe.yaml")
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func(ctx SpecContext) {
				output, err := fixture.kubectl(ctx, "delete", "application", "gitops-repository-probe", "-n", "argocd", "--ignore-not-found")
				Expect(err).NotTo(HaveOccurred(), output)
			}, NodeTimeout(time.Minute))
			output, err := fixture.kubectl(ctx, "apply", "-f", manifestPath)
			Expect(err).NotTo(HaveOccurred(), output)
			var application gitopsApplication

			// Act
			waitOutput, waitErr := fixture.kubectl(ctx, "wait", "application/gitops-repository-probe", "-n", "argocd", "--for=jsonpath={.status.reconciledAt}", "--timeout=180s")
			output, err = fixture.kubectl(ctx, "get", "application/gitops-repository-probe", "-n", "argocd", "-o", "json")

			// Assert
			Expect(waitErr).NotTo(HaveOccurred(), waitOutput)
			Expect(err).NotTo(HaveOccurred(), output)
			Expect(json.Unmarshal([]byte(output), &application)).To(Succeed())
			for _, condition := range application.Status.Conditions {
				Expect(condition.Type).NotTo(HaveSuffix("Error"), output)
			}

			Expect(application.Status.Resources).NotTo(BeEmpty(), output)
		}, SpecTimeout(4*time.Minute))
	})
})
