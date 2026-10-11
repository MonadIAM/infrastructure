package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
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
		var output bytes.Buffer

		BeforeEach(func() {
			output.Reset()
		})

		DescribeTable("[Behavior] - shows help without loading configuration",
			func(args []string) {
				// Arrange
				GinkgoT().Chdir(GinkgoT().TempDir())
				if len(args) > 1 {
					args = append(args, "--env-file", "missing.env")
				}

				// Act
				actionErr := Run(context.Background(), args, &output)

				// Assert
				Expect(actionErr).To(Succeed())
				Expect(output.String()).To(ContainSubstring("Usage:"))
			},
			Entry("[case] - no command", []string{}),
			Entry("[case] - root help", []string{"--help"}),
			Entry("[case] - Debezium help", []string{"debezium", "reconcile", "--help"}),
		)

		DescribeTable("[Behavior] - rejects invalid arguments without exposing their values",
			func(args []string) {
				// Act
				err := Run(context.Background(), args, &output)

				// Assert
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).NotTo(ContainSubstring("private-value"))
				Expect(output.String()).NotTo(ContainSubstring("private-value"))
			},
			Entry("[case] - unknown command", []string{"private-value"}),
			Entry("[case] - unknown flag", []string{"debezium", "reconcile", "--unknown=private-value"}),
			Entry("[case] - unexpected argument", []string{"config", "validate", "private-value"}),
		)

		Context("[Behavior] - configuration validation", func() {
			It("[case] - rejects a missing explicitly selected environment file", func() {
				// Arrange
				path := filepath.Join(GinkgoT().TempDir(), "missing")
				args := []string{"config", "validate", "--env-file", path}

				// Act
				err := Run(context.Background(), args, &output)

				// Assert
				Expect(err).To(MatchError(ContainSubstring("environment file")))
			})

			It("[case] - validates the example configuration without contacting HTTP services", func() {
				// Arrange
				GinkgoT().Chdir("..")
				environment, err := godotenv.Read(".env.example")
				Expect(err).NotTo(HaveOccurred())
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
					requests.Add(1)
					writer.WriteHeader(http.StatusServiceUnavailable)
				}))
				DeferCleanup(server.Close)

				overrides := map[string]string{
					"CONNECT_URL":            server.URL,
					"CLICKHOUSE_URL":         server.URL,
					"SEAWEEDFS_ENDPOINT":     server.URL,
					"VAULT_ADDR":             server.URL,
					"KAFKA_BOOTSTRAP_SERVER": "127.0.0.1:1",
					"CONNECTOR_TEMPLATE":     "messaging/debezium/outbox.connector.json",
					"CLICKHOUSE_SCHEMA_DIR":  "audit-archive/clickhouse/schema",
					"SEAWEEDFS_CONFIG_DIR":   "audit-archive/seaweedfs",
					"VAULT_POLICIES_DIR":     "vault/policies",
					"VAULT_INIT_DIR":         GinkgoT().TempDir(),
					"INFRA_TIMEOUT":          "1s",
					"DEBEZIUM_TIMEOUT":       "1s",
					"ARCHIVE_TIMEOUT":        "1s",
				}

				for key, value := range overrides {
					environment[key] = value
				}

				for key, value := range environment {
					GinkgoT().Setenv(key, value)
				}

				args := []string{
					"config", "validate",
					"--env-file", ".env.example",
					"--template", "messaging/debezium/outbox.connector.json",
					"--connect-url", server.URL,
				}

				// Act
				actionErr := Run(context.Background(), args, &output)

				// Assert
				Expect(actionErr).To(Succeed())
				Expect(output.String()).To(ContainSubstring("valid (6 connectors)"))
				Expect(requests.Load()).To(BeZero())
			})
		})
	})
})
