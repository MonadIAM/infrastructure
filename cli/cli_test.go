package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/joho/godotenv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCLI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CLI")
}

var _ = Describe("[CLI] - Commands", func() {
	Describe("[Function] - Run", func() {
		DescribeTable("[Behavior] - shows help without configuration or tools",
			func(args []string) {
				// Arrange
				GinkgoT().Chdir(GinkgoT().TempDir())
				var output bytes.Buffer

				// Act
				err := Run(context.Background(), args, &output)

				// Assert
				Expect(err).To(Succeed())
				Expect(output.String()).To(ContainSubstring("Usage:"))
			},
			Entry("[case] - root help", []string{"--help"}),
			Entry("[case] - cluster help", []string{"cluster", "start", "--help"}),
			Entry("[case] - platform help", []string{"platform", "reconcile", "--help"}),
		)

		DescribeTable("[Behavior] - rejects unknown arguments without exposing their values",
			func(args []string) {
				// Arrange
				var output bytes.Buffer

				// Act
				err := Run(context.Background(), args, &output)

				// Assert
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).NotTo(ContainSubstring("private-value"))
				Expect(output.String()).NotTo(ContainSubstring("private-value"))
			},
			Entry("[case] - unknown command", []string{"private-value", "start"}),
			Entry("[case] - unknown flag", []string{"cluster", "start", "--private-value"}),
			Entry("[case] - positional argument", []string{"cluster", "stop", "private-value"}),
		)

		Context("[Behavior] - validates input before execution", func() {
			It("[case] - validates the example without external tools", func() {
				// Arrange
				environment, readErr := godotenv.Read("../.env.example")
				Expect(readErr).To(Succeed())

				for key, value := range environment {
					GinkgoT().Setenv(key, value)
				}

				GinkgoT().Setenv("PATH", GinkgoT().TempDir())
				var output bytes.Buffer

				// Act
				err := Run(context.Background(), []string{"config", "validate", "--env-file=../.env.example"}, &output)

				// Assert
				Expect(err).To(Succeed())
				Expect(output.String()).To(Equal("Stand configuration is valid.\n"))
			})

			It("[case] - rejects unconfirmed recreation before launching tools", func() {
				// Arrange
				var output bytes.Buffer

				// Act
				err := Run(context.Background(), []string{"cluster", "recreate", "--env-file="}, &output)

				// Assert
				Expect(err).To(MatchError(ContainSubstring("--confirm-delete")))
				Expect(output.String()).To(BeEmpty())
			})

			It("[case] - an explicitly empty timeout overrides the environment and fails", func() {
				// Arrange
				GinkgoT().Setenv("INFRA_TIMEOUT", "1h")
				var output bytes.Buffer

				// Act
				err := Run(context.Background(), []string{"cluster", "stop", "--env-file=", "--timeout="}, &output)

				// Assert
				Expect(err).To(MatchError("INFRA_TIMEOUT must be a positive duration"))
			})
		})
	})
})
