package kubernetes

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

type runnerFunc func(context.Context, command.Request) (command.Result, error)

func (run runnerFunc) Run(ctx context.Context, request command.Request) (command.Result, error) {
	return run(ctx, request)
}

func TestKubernetes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kubernetes")
}

var _ = Describe("[Infrastructure] - Kubernetes", func() {
	Describe("[Method] - Client.Apply", func() {
		DescribeTable("[Behavior] - targets the selected cluster and forwards the manifest", func(serverSide bool) {
			// Arrange
			var request command.Request
			client := Client{
				Context: "stand",
				Runner: runnerFunc(func(_ context.Context, current command.Request) (command.Result, error) {
					request = current

					return command.Result{}, nil
				}),
			}
			manifest := []byte("apiVersion: v1\nkind: Namespace\nmetadata:\n  name: example\n")

			// Act
			err := client.Apply(context.Background(), manifest, serverSide)
			usesServerSide := slices.Contains(request.Args, "--server-side")

			// Assert
			Expect(err).To(Succeed())
			Expect(request.Name).To(Equal("kubectl"))
			Expect(request.Args[:2]).To(Equal([]string{"--context", "stand"}))
			Expect(request.Args).To(ContainElements("apply", "-f", "-"))
			Expect(request.Input).To(Equal(manifest))
			Expect(usesServerSide).To(Equal(serverSide))
		},
			Entry("[case] - client-side apply", false),
			Entry("[case] - server-side apply", true),
		)
	})

	Describe("[Method] - Client.WaitPods", func() {
		It("[case] - stops when pod creation fails", func() {
			// Arrange
			var requests []command.Request
			client := Client{
				Runner: runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
					requests = append(requests, request)

					return command.Result{}, errors.New("creation timed out")
				}),
			}

			// Act
			err := client.WaitPods(context.Background(), "messaging", "app=kafka", "1s")

			// Assert
			Expect(err).To(MatchError("creation timed out"))
			Expect(requests).To(HaveLen(1))
			Expect(requests[0].Args).To(ContainElement("--for=create"))
		})
	})

	Describe("[Method] - Client.Get", func() {
		It("[case] - does not disclose malformed resource contents", func() {
			// Arrange
			client := Client{
				Runner: runnerFunc(func(_ context.Context, _ command.Request) (command.Result, error) {
					return command.Result{Output: []byte("private-value")}, nil
				}),
			}
			var result map[string]any

			// Act
			err := client.Get(context.Background(), "vault", "secret/example", &result)

			// Assert
			Expect(err).To(MatchError("kubernetes.Client.Get: invalid JSON for vault/secret/example"))
		})
	})

	Describe("[Method] - Client.WaitState", func() {
		It("[case] - completes when the resource reports the expected state", func() {
			// Arrange
			client := Client{
				Runner: runnerFunc(func(_ context.Context, _ command.Request) (command.Result, error) {
					return command.Result{Output: []byte(`{"status":{"status":"Completed"}}`)}, nil
				}),
			}

			// Act
			err := client.WaitState(context.Background(), "data", "clickhouse/example", "status", "Completed", time.Second)

			// Assert
			Expect(err).To(Succeed())
		})

		It("[case] - preserves the cause and resource identity of a failed read", func() {
			// Arrange
			failure := errors.New("permission denied")
			calls := 0
			client := Client{
				Runner: runnerFunc(func(_ context.Context, _ command.Request) (command.Result, error) {
					calls++

					return command.Result{}, failure
				}),
			}

			// Act
			err := client.WaitState(context.Background(), "data", "clickhouse/example", "status", "Completed", time.Second)

			// Assert
			Expect(err).To(MatchError("kubernetes.Client.WaitState: waiting for data/clickhouse/example: permission denied"))
			Expect(err).To(MatchError(failure))
			Expect(calls).To(Equal(1))
		})
	})
})
