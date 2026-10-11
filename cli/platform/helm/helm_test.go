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
	Describe("[Function] - Execute", func() {
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
