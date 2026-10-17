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

var _ = Describe("[Infrastructure] - PostgreSQL", func() {
	var environment config.Environment

	BeforeEach(func() {
		var err error
		environment, err = godotenv.Read("../../../.env.example")
		Expect(err).To(Succeed())
	})

	Describe("[Method] - Platform.applyClusterResources", func() {
		It("[case] - establishes monitoring CRDs before applying data resources", func() {
			// Arrange
			var commands []string
			runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
				commands = append(commands, strings.Join(request.Args, " "))

				return command.Result{Output: []byte("rendered resources")}, nil
			})
			platform := New(environment, runner, io.Discard)
			platform.Helm = helm.Client{
				Runner:   runner,
				Attempts: 1,
				Output:   io.Discard,
			}

			// Act
			err := platform.applyClusterResources(context.Background())
			joined := strings.Join(commands, "\n")
			monitor := strings.Index(joined, "crd/podmonitors.monitoring.coreos.com")
			postgres := strings.Index(joined, "apply -k platform/postgresql/resources")

			// Assert
			Expect(err).To(Succeed())
			Expect(monitor).To(BeNumerically(">=", 0))
			Expect(postgres).To(BeNumerically(">", monitor))
		})
	})

	Describe("[Behavior] - readiness failures", func() {
		for index, cluster := range postgresqlClusters {
			It("[case] - stops at "+cluster, func() {
				// Arrange
				var waited []string
				failure := errors.New("PostgreSQL unavailable")
				runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
					for _, argument := range request.Args {
						if strings.HasPrefix(argument, "cluster.postgresql.cnpg.io/") {
							waited = append(waited, strings.TrimPrefix(argument, "cluster.postgresql.cnpg.io/"))
							if argument == "cluster.postgresql.cnpg.io/"+cluster {
								return command.Result{}, failure
							}
						}
					}

					return command.Result{Output: []byte("rendered resources")}, nil
				})
				platform := New(environment, runner, io.Discard)
				platform.Helm = helm.Client{Runner: runner, Attempts: 1, Output: io.Discard}

				// Act
				err := platform.Clusters(context.Background())

				// Assert
				Expect(err).To(MatchError(failure))
				Expect(waited).To(Equal(postgresqlClusters[:index+1]))
			})
		}
	})
})
