package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

func validSettings() Debezium {
	return Debezium{
		ConnectURL:                  "http://kafka-connect:8083",
		TemplatePath:                "connector.json",
		Timeout:                     time.Minute,
		RequestTimeout:              time.Second,
		RetryInterval:               time.Second,
		RealmPartitions:             4,
		PositionPartitions:          2,
		ProjectionJournalPartitions: 8,
		Databases: []Database{
			{
				Service:     "identity-service",
				Host:        "identity-postgresql",
				Name:        "identity",
				User:        "test",
				Password:    "test-only",
				Slot:        "identity_slot",
				Publication: "identity_publication",
			},
		},
	}
}

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Configuration")
}

var _ = Describe("[Config] - Debezium", func() {
	var values map[string]string

	BeforeEach(func() {
		var err error
		values, err = godotenv.Read("../../.env.example")
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("[Function] - ParseDebezium", func() {
		Context("[Behavior] - defaults and service mapping", func() {
			It("[case] - requires only the settings used by Debezium", func() {
				// Arrange
				for key := range values {
					if !strings.Contains(key, "_POSTGRES_") && !strings.HasSuffix(key, "_PARTITIONS") {
						delete(values, key)
					}
				}

				// Act
				settings, err := ParseDebezium(values)

				// Assert
				Expect(err).NotTo(HaveOccurred())
				Expect(settings.Databases).To(ContainElements(
					MatchFields(IgnoreExtras, Fields{"Service": Equal("identity-service"), "Host": Equal("identity-postgresql")}),
					MatchFields(IgnoreExtras, Fields{"Service": Equal("notification-service"), "Host": Equal("notification-postgresql")}),
				))
				Expect(settings.Timeout).To(Equal(2 * time.Minute))
			})
		})

		DescribeTable("[Behavior] - rejects malformed numeric settings without exposing their values",
			func(key string) {
				// Arrange
				values[key] = "private-value"

				// Act
				_, err := ParseDebezium(values)

				// Assert
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(key))
				Expect(err.Error()).NotTo(ContainSubstring("private-value"))
			},
			Entry("[case] - realm partitions", "REALM_PARTITIONS"),
			Entry("[case] - position partitions", "POSITION_PARTITIONS"),
			Entry("[case] - projection journal partitions", "PROJECTION_JOURNAL_PARTITIONS"),
			Entry("[case] - execution deadline", "DEBEZIUM_TIMEOUT"),
			Entry("[case] - request deadline", "DEBEZIUM_REQUEST_TIMEOUT"),
			Entry("[case] - retry delay", "DEBEZIUM_RETRY_INTERVAL"),
		)
	})

	Describe("[Function] - Load", func() {
		It("[case] - prefers flags to environment and environment to dotenv without changing the process", func() {
			// Arrange
			GinkgoT().Setenv("CONNECT_URL", "http://environment:8083")
			GinkgoT().Setenv("REALM_PARTITIONS", "6")

			path := filepath.Join(GinkgoT().TempDir(), ".env")
			contents := "CONNECT_URL='http://file:8083'\nREALM_PARTITIONS=2\n"
			Expect(os.WriteFile(path, []byte(contents), 0600)).To(Succeed())

			// Act
			environment, err := Load(path, map[string]string{
				"CONNECT_URL": "http://flag:8083",
			})
			processURL := os.Getenv("CONNECT_URL")

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(environment).To(HaveKeyWithValue("CONNECT_URL", "http://flag:8083"))
			Expect(environment).To(HaveKeyWithValue("REALM_PARTITIONS", "6"))
			Expect(processURL).To(Equal("http://environment:8083"))
		})

		It("[case] - does not replace an explicitly empty environment value with a dotenv value", func() {
			// Arrange
			GinkgoT().Setenv("IDENTITY_POSTGRES_PASSWORD", "")

			// Act
			environment, err := Load("../../.env.example", nil)

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(environment).To(HaveKeyWithValue("IDENTITY_POSTGRES_PASSWORD", ""))
		})

		It("[case] - does not expose dotenv contents on a parsing error", func() {
			// Arrange
			path := filepath.Join(GinkgoT().TempDir(), ".env")
			Expect(os.WriteFile(path, []byte("KEY=\"private-value"), 0600)).To(Succeed())

			// Act
			_, err := Load(path, nil)

			// Assert
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).NotTo(ContainSubstring("private-value"))
		})
	})

	Describe("[Method] - Debezium.Validate", func() {
		DescribeTable("[Behavior] - rejects invalid configuration",
			func(change func(*Debezium), message string) {
				// Arrange
				settings := validSettings()
				change(&settings)

				// Act
				err := settings.Validate()

				// Assert
				Expect(err).To(MatchError(ContainSubstring(message)))
			},
			Entry("[case] - credentials in URL", func(settings *Debezium) {
				settings.ConnectURL = "http://test:secret@connect:8083"
			}, "credentials"),
			Entry("[case] - unsupported URL scheme", func(settings *Debezium) {
				settings.ConnectURL = "file://connect/config"
			}, "HTTP or HTTPS"),
			Entry("[case] - query in URL", func(settings *Debezium) {
				settings.ConnectURL += "?token=secret"
			}, "query"),
			Entry("[case] - fragment in URL", func(settings *Debezium) {
				settings.ConnectURL += "#private-value"
			}, "fragment"),
			Entry("[case] - missing host in URL", func(settings *Debezium) {
				settings.ConnectURL = "http:///connect"
			}, "HTTP(S) URL"),
			Entry("[case] - missing template", func(settings *Debezium) {
				settings.TemplatePath = ""
			}, "CONNECTOR_TEMPLATE"),
			Entry("[case] - zero deadline", func(settings *Debezium) {
				settings.Timeout = 0
			}, "must be positive"),
			Entry("[case] - zero partition count", func(settings *Debezium) {
				settings.PositionPartitions = 0
			}, "partition counts"),
			Entry("[case] - no databases", func(settings *Debezium) {
				settings.Databases = nil
			}, "at least one"),
			Entry("[case] - missing database password", func(settings *Debezium) {
				settings.Databases[0].Password = ""
			}, "outbox password is required"),
			Entry("[case] - duplicate service", func(settings *Debezium) {
				settings.Databases = append(settings.Databases, settings.Databases[0])
			}, "duplicate outbox service"),
			Entry("[case] - invalid service name", func(settings *Debezium) {
				settings.Databases[0].Service = "bad/name"
			}, "path segment"),
		)
	})
})
