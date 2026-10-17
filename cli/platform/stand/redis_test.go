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

var _ = Describe("[Infrastructure] - Redis", func() {
	var environment config.Environment

	BeforeEach(func() {
		var err error
		environment, err = godotenv.Read("../../../.env.example")
		Expect(err).To(Succeed())
	})

	Describe("[Method] - Platform.installRedis", func() {
		It("[case] - stops before installing the next release when Redis installation fails", func() {
			// Arrange
			var requests []command.Request
			failure := errors.New("Redis chart download failed")
			runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
				requests = append(requests, request)

				if request.Name == "helm" {
					return command.Result{}, failure
				}

				return command.Result{}, nil
			})
			platform := New(environment, runner, io.Discard)
			platform.Helm = helm.Client{Runner: runner, Attempts: 1, Output: io.Discard}

			// Act
			err := platform.installRedis(context.Background())

			// Assert
			Expect(err).To(MatchError(failure))
			Expect(requests).To(HaveLen(3))
			Expect(requests[2].Name).To(Equal("helm"))
			Expect(requests[2].Args[:4]).To(Equal([]string{"upgrade", "--install", "access-control-redis", environment["REDIS_CHART"]}))
		})
	})

	Describe("[Behavior] - per-service installation", func() {
		It("[case] - waits for credentials and TLS before installing each upstream release", func() {
			// Arrange
			var requests []command.Request
			runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
				requests = append(requests, request)
				return command.Result{}, nil
			})
			platform := New(environment, runner, io.Discard)
			platform.Helm = helm.Client{Runner: runner, Attempts: 1, Output: io.Discard}

			// Act
			err := platform.installRedis(context.Background())

			// Assert
			Expect(err).To(Succeed())
			Expect(requests).To(HaveLen(3 * len(redisClusters)))
			for index, cluster := range redisClusters {
				certificate := requests[index*3]
				secret := requests[index*3+1]
				release := requests[index*3+2]
				Expect(certificate.Name).To(Equal("kubectl"))
				Expect(certificate.Args).To(ContainElement("certificate/" + cluster + "-tls"))
				Expect(secret.Args).To(ContainElement("secret/" + cluster + "-auth"))
				Expect(release.Name).To(Equal("helm"))
				Expect(release.Args[:4]).To(Equal([]string{"upgrade", "--install", cluster, environment["REDIS_CHART"]}))
				Expect(strings.Join(release.Args, " ")).To(ContainSubstring("--version " + environment["REDIS_CHART_VERSION"]))
				Expect(release.Args).To(ContainElement("platform/redis/helm/services/" + strings.TrimSuffix(cluster, "-redis") + ".yaml"))
			}
		})
	})

	Describe("[Behavior] - installation prerequisites", func() {
		DescribeTable("[case] - stops before Helm when a dependency is unavailable", func(resource string, calls int) {
			// Arrange
			var requests []command.Request
			failure := errors.New("dependency unavailable")
			runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
				requests = append(requests, request)
				if strings.Contains(strings.Join(request.Args, " "), resource) {
					return command.Result{}, failure
				}

				return command.Result{}, nil
			})
			platform := New(environment, runner, io.Discard)
			platform.Helm = helm.Client{Runner: runner, Attempts: 1, Output: io.Discard}

			// Act
			err := platform.installRedis(context.Background())

			// Assert
			Expect(err).To(MatchError(failure))
			Expect(requests).To(HaveLen(calls))
			for _, request := range requests {
				Expect(request.Name).To(Equal("kubectl"))
			}
		}, Entry("certificate", "certificate/access-control-redis-tls", 1), Entry("password", "secret/access-control-redis-auth", 2))
	})
})
