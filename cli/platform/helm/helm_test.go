package helm

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

type writerFunc func([]byte) (int, error)

type runnerFunc func(context.Context, command.Request) (command.Result, error)

func (run runnerFunc) Run(ctx context.Context, request command.Request) (command.Result, error) {
	return run(ctx, request)
}

func (write writerFunc) Write(data []byte) (int, error) {
	return write(data)
}

func TestHelm(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Helm")
}

var _ = Describe("[Infrastructure] - Helm", func() {
	Describe("[Method] - Client.Install", func() {
		DescribeTable("[Behavior] - supports remote and local charts", func(chart, version string) {
			// Arrange
			var arguments []string
			client := Client{
				Attempts: 1,
				Output:   io.Discard,
				Runner: runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
					arguments = request.Args

					return command.Result{}, nil
				}),
			}

			// Act
			err := client.Install(context.Background(), Release{
				Name: "example", Chart: chart, Version: version, Namespace: "data", Values: "values.yaml",
			})

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(arguments[:4]).To(Equal([]string{"upgrade", "--install", "example", chart}))
			if version == "" {
				Expect(arguments).NotTo(ContainElement("--version"))
			} else {
				Expect(arguments).To(ContainElement("--version"))
				Expect(arguments).To(ContainElement(version))
			}
		},
			Entry("local chart", "./example-chart", ""),
			Entry("remote chart", "example/redis", "28.0.0"),
		)
	})

	Describe("[Method] - Client.Execute", func() {
		It("[case] - discards partial manifests from failed attempts", func() {
			// Arrange
			attempts := 0
			var requests []command.Request
			client := Client{
				Attempts: 2,
				Output:   io.Discard,
				Runner: runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
					requests = append(requests, request)
					attempts++
					if attempts == 1 {
						return command.Result{Output: []byte("incomplete document")}, errors.New("download failed")
					} else {
						return command.Result{Output: []byte("complete document")}, nil
					}
				}),
			}

			// Act
			output, err := client.Execute(context.Background(), true, "show", "crds", "chart")

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(string(output)).To(Equal("complete document"))
			Expect(attempts).To(Equal(2))
			Expect(requests).To(HaveEach(HaveField("Capture", BeTrue())))
		})

		It("[case] - cancels while waiting between failed attempts", func() {
			// Arrange
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var attempts int
			output := writerFunc(func(data []byte) (int, error) {
				time.AfterFunc(10*time.Millisecond, cancel)

				return len(data), nil
			})
			client := Client{
				Attempts: 5,
				Delay:    time.Hour,
				MaxDelay: time.Hour,
				Output:   output,
				Runner: runnerFunc(func(_ context.Context, _ command.Request) (command.Result, error) {
					attempts++

					return command.Result{}, errors.New("network failure")
				}),
			}

			// Act
			started := time.Now()
			_, err := client.Execute(ctx, true, "show", "crds", "chart")
			elapsed := time.Since(started)

			// Assert
			Expect(err).To(MatchError(context.Canceled))
			Expect(elapsed).To(BeNumerically("<", time.Second))
			Expect(attempts).To(Equal(1))
		})

		It("[case] - returns the final failure after exhausting retries", func() {
			// Arrange
			attempts := 0
			failure := errors.New("repository unavailable")
			client := Client{
				Attempts: 3,
				Output:   io.Discard,
				Runner: runnerFunc(func(_ context.Context, _ command.Request) (command.Result, error) {
					attempts++

					return command.Result{Output: []byte("partial")}, failure
				}),
			}

			// Act
			output, err := client.Execute(context.Background(), true, "show", "crds", "chart")

			// Assert
			Expect(err).To(MatchError(failure))
			Expect(output).To(BeEmpty())
			Expect(attempts).To(Equal(3))
		})
	})
})
