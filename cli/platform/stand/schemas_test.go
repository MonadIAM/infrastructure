package stand

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/joho/godotenv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

var _ = Describe("[Infrastructure] - Shared schemas", func() {
	var environment config.Environment
	var requests []command.Request
	var jobState string
	var failedStep string
	var failure error
	var platform Platform

	BeforeEach(func() {
		var err error
		environment, err = godotenv.Read("../../../.env.example")
		Expect(err).To(Succeed())
		environment["SHARED_SCHEMAS_IMAGE"] = "ghcr.io/monadiam/shared-schemas:3.0.1@sha256:" + strings.Repeat("a", 64)
		requests = nil
		failedStep = ""
		failure = errors.New("publication interrupted")
		jobState = `{"status":{"conditions":[{"type":"Complete","status":"True"}]}}`
		runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
			requests = append(requests, request)
			arguments := request.Name + " " + strings.Join(request.Args, " ")
			if failedStep != "" && strings.Contains(arguments, failedStep) {
				return command.Result{}, failure
			}

			if strings.Contains(arguments, "kustomize") {
				return command.Result{Output: []byte("image: ${SHARED_SCHEMAS_IMAGE}")}, nil
			} else if strings.Contains(arguments, "get job/shared-schemas") {
				return command.Result{Output: []byte(jobState)}, nil
			}

			return command.Result{}, nil
		})
		platform = New(environment, runner, io.Discard)
	})

	It("[case] - prepares images before platform installation and publishes after Apicurio", func() {
		// Arrange
		var names []string

		// Act
		for _, phase := range platform.phases() {
			names = append(names, phase.Name)
		}

		// Assert
		Expect(names[0]).To(Equal("images"))
		Expect(strings.Join(names, ",")).To(ContainSubstring("apicurio,schemas,seaweedfs"))
	})

	It("[case] - prepares the pinned image and runs the publication Job", func() {
		// Arrange
		image := environment["SHARED_SCHEMAS_IMAGE"]
		imported := strings.Replace(image, ":3.0.1@", "@", 1)

		// Act
		err := platform.Reconcile(context.Background(), "schemas")

		// Assert
		Expect(err).To(Succeed())
		Expect(requests).To(HaveLen(8))
		Expect(requests[0].Name).To(Equal("docker"))
		Expect(requests[0].Args).To(Equal([]string{"pull", imported}))
		Expect(requests[1].Name).To(Equal("k3d"))
		Expect(requests[1].Args).To(Equal([]string{"image", "import", imported, "--cluster", environment["CLUSTER"]}))
		Expect(strings.Join(requests[2].Args, " ")).To(ContainSubstring("rollout status deployment/schema-registry-main"))
		Expect(requests[3].Args[2:]).To(Equal([]string{"kustomize", "platform/apicurio/schemas"}))
		Expect(requests[4].Args[2:]).To(Equal([]string{"delete", "job", "shared-schemas", "-n", messagingNamespace, "--ignore-not-found", "--wait=true", "--timeout=60s"}))
		Expect(string(requests[5].Input)).To(Equal("image: " + image))
		Expect(requests[6].Args[2:]).To(Equal([]string{"get", "job/shared-schemas", "-n", messagingNamespace, "-o", "json", "--request-timeout=10s"}))
		Expect(requests[7].Args[2:]).To(Equal([]string{"logs", "job/shared-schemas", "-n", messagingNamespace, "--tail=100"}))
		for _, request := range requests[2:] {
			Expect(request.Name).To(Equal("kubectl"))
			Expect(request.Args[:2]).To(Equal([]string{"--context", environment["KUBE_CONTEXT"]}))
		}
	})

	DescribeTable("[case] - rejects images without a valid digest before any command", func(image string) {
		// Arrange
		environment["SHARED_SCHEMAS_IMAGE"] = image

		// Act
		err := platform.Images(context.Background())

		// Assert
		Expect(err).To(MatchError(ContainSubstring("pinned to a sha256 digest")))
		Expect(requests).To(BeEmpty())
	},
		Entry("missing image", ""),
		Entry("mutable tag", "ghcr.io/monadiam/shared-schemas:3.0.1"),
		Entry("short digest", "ghcr.io/monadiam/shared-schemas@sha256:abc"),
		Entry("manifest injection", "image\nkey: value@sha256:"+strings.Repeat("a", 64)),
	)

	DescribeTable("[case] - imports by digest while preserving the registry port", func(reference, imported string) {
		// Arrange
		digest := "@sha256:" + strings.Repeat("a", 64)
		environment["SHARED_SCHEMAS_IMAGE"] = reference + digest

		// Act
		err := platform.Images(context.Background())

		// Assert
		Expect(err).To(Succeed())
		Expect(requests[0].Args).To(Equal([]string{"pull", imported + digest}))
		Expect(requests[1].Args[2]).To(Equal(imported + digest))
	},
		Entry("digest only", "ghcr.io/monadiam/shared-schemas", "ghcr.io/monadiam/shared-schemas"),
		Entry("registry port and tag", "localhost:5000/shared-schemas:3.0.1", "localhost:5000/shared-schemas"),
		Entry("registry port without tag", "localhost:5000/shared-schemas", "localhost:5000/shared-schemas"),
	)

	DescribeTable("[case] - stops at the failed preparation or publication step", func(step string, count int) {
		// Arrange
		failedStep = step

		// Act
		err := platform.Reconcile(context.Background(), "schemas")

		// Assert
		Expect(err).To(MatchError(ContainSubstring(failure.Error())))
		Expect(requests).To(HaveLen(count))
	},
		Entry("download", "docker pull", 1),
		Entry("import", "k3d image import", 2),
		Entry("registry readiness", "rollout status", 3),
		Entry("render", "kustomize", 4),
		Entry("old Job deletion", "delete job", 5),
		Entry("apply", "apply -f", 6),
	)

	DescribeTable("[case] - reports terminal Job failure immediately with diagnostics", func(condition string) {
		// Arrange
		jobState = `{"status":{"conditions":[{"type":"` + condition + `","status":"True","reason":"BackoffLimitExceeded","message":"Container failed"}]}}`

		// Act
		err := platform.Schemas(context.Background())

		// Assert
		Expect(err).To(MatchError(ContainSubstring("BackoffLimitExceeded: Container failed")))
		Expect(requests).To(HaveLen(8))
		Expect(strings.Join(requests[5].Args, " ")).To(ContainSubstring("describe job shared-schemas"))
		Expect(strings.Join(requests[6].Args, " ")).To(ContainSubstring("describe pods"))
		Expect(strings.Join(requests[7].Args, " ")).To(ContainSubstring("logs job/shared-schemas"))
	},
		Entry("Failed", "Failed"),
		Entry("FailureTarget", "FailureTarget"),
	)

	It("[case] - preserves the polling error when diagnostics also fail", func() {
		// Arrange
		jobState = "invalid JSON"
		failedStep = "describe"

		// Act
		err := platform.Schemas(context.Background())

		// Assert
		Expect(err).To(MatchError(ContainSubstring("invalid JSON")))
		Expect(strings.Join(requests[len(requests)-1].Args, " ")).To(ContainSubstring("logs job/shared-schemas"))
	})

	It("[case] - respects cancellation while the Job is pending", func() {
		// Arrange
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Act
		err := platform.waitSchemas(ctx)

		// Assert
		Expect(errors.Is(err, context.Canceled)).To(BeTrue())
		Expect(requests).To(BeEmpty())
	})

	It("[case] - replaces the previous Job on each reconciliation", func() {
		// Arrange
		var deletions, applications int

		// Act
		Expect(platform.Schemas(context.Background())).To(Succeed())
		Expect(platform.Schemas(context.Background())).To(Succeed())
		for _, request := range requests {
			arguments := strings.Join(request.Args, " ")
			if strings.Contains(arguments, "delete job shared-schemas") {
				deletions++
			} else if strings.Contains(arguments, "apply -f") {
				applications++
			}
		}

		// Assert
		Expect(deletions).To(Equal(2))
		Expect(applications).To(Equal(2))
	})
})
