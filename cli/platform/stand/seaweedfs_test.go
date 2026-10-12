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
	"github.com/MonadIAM/infrastructure/cli/platform/helm"
)

var _ = Describe("[Infrastructure] - SeaweedFS deployment", func() {
	var environment config.Environment

	BeforeEach(func() {
		var err error
		environment, err = godotenv.Read("../../../.env.example")
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("[Method] - Platform.SeaweedFS", func() {
		Context("[Behavior] - deployment ordering", func() {
			It("[case] - waits for credentials and installs storage before importing and running the bootstrap image", func() {
				// Arrange
				var requests []string
				runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
					requests = append(requests, request.Name+" "+strings.Join(request.Args, " "))

					return seaweedFSResponse(request), nil
				})
				platform := New(environment, runner, io.Discard)
				platform.Helm = helm.Client{Runner: runner, Attempts: 1, Output: io.Discard}

				// Act
				err := platform.SeaweedFS(context.Background())
				calls := strings.Join(requests, "\n")

				// Assert
				Expect(err).To(Succeed())
				Expect(strings.Index(calls, "upgrade --install seaweedfs-main")).To(BeNumerically(">", strings.Index(calls, "secret/seaweedfs-main-admin")))
				Expect(strings.Index(calls, "image import")).To(BeNumerically(">", strings.Index(calls, "upgrade --install seaweedfs-main")))
				Expect(strings.Index(calls, "delete job seaweedfs-main-bootstrap")).To(BeNumerically(">", strings.Index(calls, "image import")))
				Expect(requests[len(requests)-1]).To(ContainSubstring("--for=condition=Complete job/seaweedfs-main-bootstrap"))
				Expect(calls).To(ContainSubstring("monadiam/stand-cli:" + strings.Repeat("a", 64)))
			})
		})

		DescribeTable("[Behavior] - stops after a failed deployment stage", func(failure, forbidden string) {
			// Arrange
			var requests []string
			runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
				arguments := strings.Join(request.Args, " ")
				requests = append(requests, arguments)
				if strings.Contains(arguments, failure) {
					return command.Result{}, errors.New("stage failed")
				}

				return seaweedFSResponse(request), nil
			})
			platform := New(environment, runner, io.Discard)
			platform.Helm = helm.Client{Runner: runner, Attempts: 1, Output: io.Discard}

			// Act
			err := platform.SeaweedFS(context.Background())

			// Assert
			Expect(err).To(HaveOccurred())
			Expect(strings.Join(requests, "\n")).NotTo(ContainSubstring(forbidden))
		},
			Entry("[case] - unavailable credentials", "get vaultstaticsecrets.", "upgrade --install seaweedfs-main"),
			Entry("[case] - failed chart install", "upgrade --install seaweedfs-main", "build --file"),
			Entry("[case] - failed image build", "build --file", "image import"),
			Entry("[case] - failed image import", "image import", "delete job"),
			Entry("[case] - failed render", "kustomize platform/seaweedfs/bootstrap", "delete job"),
			Entry("[case] - failed previous job deletion", "delete job", "--for=condition=Complete"),
		)
	})

	Describe("[Method] - Platform.seaweedFSManifest", func() {
		It("[case] - keeps numeric configuration values as YAML strings", func() {
			// Arrange
			platform := Platform{Environment: environment}
			manifest := "retention: ${SEAWEEDFS_AUDIT_LOG_RETENTION_DAYS}\nregion: \"${SEAWEEDFS_REGION}\""

			// Act
			rendered, err := platform.seaweedFSManifest(manifest, "monadiam/stand-cli:test")

			// Assert
			Expect(err).To(Succeed())
			Expect(rendered).To(Equal("retention: \"1095\"\nregion: \"us-east-1\""))
		})

		It("[case] - rejects multiline substitutions before applying resources", func() {
			// Arrange
			environment["SEAWEEDFS_REGION"] = "us-east-1\ninjected: true"
			platform := Platform{Environment: environment}

			// Act
			_, err := platform.seaweedFSManifest("region: ${SEAWEEDFS_REGION}", "monadiam/stand-cli:test")

			// Assert
			Expect(err).To(MatchError("stand.Platform.seaweedFSManifest: invalid SEAWEEDFS_REGION for SeaweedFS bootstrap"))
		})
	})
})
