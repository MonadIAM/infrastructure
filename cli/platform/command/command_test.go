package command

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/MonadIAM/infrastructure/cli/config"
)

func TestCommandProcess(t *testing.T) {
	switch os.Getenv("INFRA_TEST_PROCESS") {
	case "echo":
		_, _ = io.Copy(os.Stdout, os.Stdin)
		_, _ = fmt.Fprint(os.Stderr, "private diagnostic")
		os.Exit(23)
	case "wait":
		time.Sleep(time.Minute)
		os.Exit(0)
	}
}

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Command execution")
}

var _ = Describe("[Infrastructure] - Command", func() {
	Describe("[Function] - Run", func() {
		It("[case] - captures secrets without printing them in output or errors", func() {
			// Arrange
			path, err := os.Executable()
			Expect(err).To(Succeed())
			var output bytes.Buffer
			executor := Executor{
				Environment: config.Environment{
					"INFRA_TEST_PROCESS": "echo",
					"GOCOVERDIR":         GinkgoT().TempDir(),
				},
				Output: &output,
				Errors: &output,
			}

			// Act
			result, err := executor.Run(context.Background(), Request{
				Name:      path,
				Args:      []string{"-test.run=TestCommandProcess"},
				Input:     []byte("private input"),
				Sensitive: true,
			})

			// Assert
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).NotTo(ContainSubstring("private"))
			Expect(output.String()).To(BeEmpty())
			Expect(result.ExitCode).To(Equal(23))
			Expect(string(result.Output)).To(Equal("private input"))
			Expect(string(result.Stderr)).To(Equal("private diagnostic"))
		})

		It("[case] - terminates a subprocess on deadline", func() {
			// Arrange
			path, err := os.Executable()
			Expect(err).To(Succeed())
			executor := Executor{
				Environment: config.Environment{"INFRA_TEST_PROCESS": "wait"},
				Output:      io.Discard,
				Errors:      io.Discard,
			}
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			// Act
			_, err = executor.Run(ctx, Request{
				Name: path,
				Args: []string{"-test.run=TestCommandProcess"},
			})

			// Assert
			Expect(err).To(MatchError(context.DeadlineExceeded))
		})
	})

	Describe("[Function] - Poll", func() {
		It("[case] - does not retry an explicit error", func() {
			// Arrange
			calls := 0
			ready := func(context.Context) (bool, error) {
				calls++

				return false, io.ErrUnexpectedEOF
			}

			// Act
			err := Poll(context.Background(), time.Minute, time.Hour, ready)

			// Assert
			Expect(err).To(MatchError(io.ErrUnexpectedEOF))
			Expect(calls).To(Equal(1))
		})

		It("[case] - stops polling when the resource becomes ready", func() {
			// Arrange
			calls := 0
			ready := func(context.Context) (bool, error) {
				calls++

				return calls == 3, nil
			}

			// Act
			err := Poll(context.Background(), time.Second, time.Nanosecond, ready)

			// Assert
			Expect(err).To(Succeed())
			Expect(calls).To(Equal(3))
		})

		It("[case] - does not invoke the check after cancellation", func() {
			// Arrange
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			calls := 0
			ready := func(context.Context) (bool, error) {
				calls++

				return false, nil
			}

			// Act
			err := Poll(ctx, time.Minute, time.Hour, ready)

			// Assert
			Expect(err).To(MatchError(context.Canceled))
			Expect(calls).To(BeZero())
		})

		It("[case] - interrupts the polling interval when the deadline expires", func() {
			// Arrange
			calls := 0
			ready := func(context.Context) (bool, error) {
				calls++

				return false, nil
			}

			// Act
			err := Poll(context.Background(), 100*time.Millisecond, time.Hour, ready)

			// Assert
			Expect(err).To(MatchError(context.DeadlineExceeded))
			Expect(calls).To(Equal(1))
		})
	})
})
