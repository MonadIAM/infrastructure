package debezium

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/MonadIAM/infrastructure/cli/config"
)

func testSettings() config.Debezium {
	return config.Debezium{
		ConnectURL:                  "http://connect:8083",
		TemplatePath:                "../../../messaging/debezium/outbox.connector.json",
		Timeout:                     time.Second,
		RequestTimeout:              200 * time.Millisecond,
		RetryInterval:               time.Millisecond,
		RealmPartitions:             4,
		PositionPartitions:          2,
		ProjectionJournalPartitions: 8,
		Databases: []config.Database{
			{
				Service:     "identity-service",
				Host:        "identity-postgresql",
				Name:        "identity",
				User:        "test",
				Password:    "private\"\\\n${REALM_PARTITIONS}",
				Slot:        "identity_slot",
				Publication: "identity_publication",
			},
		},
	}
}

func TestDebezium(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Debezium")
}

var _ = Describe("[Infrastructure] - Debezium", func() {
	Describe("[Method] - Service.Build", func() {
		It("[case] - preserves routing, static settings and credentials with special characters", func() {
			// Arrange
			settings := testSettings()
			template, err := os.ReadFile(settings.TemplatePath)
			Expect(err).NotTo(HaveOccurred())

			var original map[string]string
			Expect(json.Unmarshal(template, &original)).To(Succeed())

			// Act
			connectors, err := New(settings).Build(template)

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(connectors).To(HaveLen(1))

			Expect(connectors[0].Config).To(HaveKeyWithValue("database.password", settings.Databases[0].Password))
			Expect(connectors[0].Config).To(HaveKeyWithValue("transforms.outbox.route.topic.replacement", "${routedByValue}"))
			Expect(connectors[0].Config).To(HaveKeyWithValue("transforms.realmPartition.partition.topic.num", "4"))
			Expect(connectors[0].Config).To(HaveKeyWithValue("transforms.positionPartition.partition.topic.num", "2"))
			Expect(connectors[0].Config).To(HaveKeyWithValue("transforms.partition.partition.topic.num", "8"))

			Expect(connectors[0].Config).To(HaveLen(len(original)))

			for key, value := range original {
				if !strings.Contains(value, "${") {
					Expect(connectors[0].Config).To(HaveKeyWithValue(key, value))
				}
			}
		})
	})

	Describe("[Method] - Service.Reconcile", func() {
		var settings config.Debezium
		var output bytes.Buffer

		BeforeEach(func() {
			settings = testSettings()
			output.Reset()
		})

		Context("[Behavior] - registration retries", func() {
			It("[case] - retries registration after a rebalance conflict", func(ctx SpecContext) {
				// Arrange
				var puts atomic.Int32
				var mutex sync.Mutex
				var requests []recordedRequest
				server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					if request.Method == http.MethodGet {
						writer.WriteHeader(http.StatusOK)

						return
					}

					record := recordedRequest{
						method:      request.Method,
						path:        request.URL.Path,
						contentType: request.Header.Get("Content-Type"),
					}
					record.decodeErr = json.NewDecoder(request.Body).Decode(&record.body)

					mutex.Lock()
					requests = append(requests, record)
					mutex.Unlock()

					if puts.Add(1) == 1 {
						writer.WriteHeader(http.StatusConflict)
					}

					_, _ = io.WriteString(writer, "private-response")
				}))
				DeferCleanup(server.Close)
				settings.ConnectURL = server.URL

				// Act
				actionErr := New(settings).Reconcile(ctx, &output)
				mutex.Lock()
				captured := append([]recordedRequest(nil), requests...)
				mutex.Unlock()

				// Assert
				Expect(actionErr).To(Succeed())
				Expect(puts.Load()).To(Equal(int32(2)))

				Expect(captured).To(HaveLen(2))

				for _, request := range captured {
					Expect(request.method).To(Equal(http.MethodPut))
					Expect(request.path).To(Equal("/connectors/identity-service-outbox-connector/config"))
					Expect(request.contentType).To(Equal("application/json"))
					Expect(request.decodeErr).NotTo(HaveOccurred())
					Expect(request.body).To(HaveKeyWithValue("database.password", settings.Databases[0].Password))
				}

				Expect(output.String()).NotTo(ContainSubstring("private"))
			})
		})

		DescribeTable("[Behavior] - validates every connector before making requests",
			func(invalidTemplate bool, message string) {
				// Arrange
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
					requests.Add(1)
				}))
				DeferCleanup(server.Close)
				settings.ConnectURL = server.URL

				if invalidTemplate {
					settings.TemplatePath = filepath.Join(GinkgoT().TempDir(), "connector.json")
					template := []byte(`{"database.password":"${UNKNOWN}"}`)
					Expect(os.WriteFile(settings.TemplatePath, template, 0600)).To(Succeed())
				} else {
					settings.Databases = append(settings.Databases, config.Database{Service: "broken-service"})
				}

				// Act
				actionErr := New(settings).Reconcile(context.Background(), &output)

				// Assert
				Expect(actionErr).To(MatchError(ContainSubstring(message)))
				Expect(requests.Load()).To(BeZero())
			},
			Entry("[case] - missing settings in the last database", false, "broken-service: outbox host is required"),
			Entry("[case] - unknown template placeholder", true, "unsupported placeholder"),
		)

		Context("[Behavior] - rejected registration", func() {
			It("[case] - stops before the next connector on a permanent error and hides the response body", func(ctx SpecContext) {
				// Arrange
				var puts atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					if request.Method == http.MethodPut {
						puts.Add(1)
						writer.WriteHeader(http.StatusBadRequest)
						_, _ = io.WriteString(writer, "private-password")
					}
				}))
				DeferCleanup(server.Close)
				settings.ConnectURL = server.URL
				second := settings.Databases[0]
				second.Service = "template-service"
				settings.Databases = append(settings.Databases, second)

				// Act
				err := New(settings).Reconcile(ctx, &output)

				// Assert
				Expect(err).To(MatchError(ContainSubstring("status 400")))
				Expect(err.Error()).NotTo(ContainSubstring("private"))
				Expect(puts.Load()).To(Equal(int32(1)))
			})
		})

		DescribeTable("[Behavior] - does not retry permanent readiness errors or follow redirects",
			func(status int) {
				// Arrange
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					requests.Add(1)
					writer.Header().Set("Location", "/elsewhere")
					writer.WriteHeader(status)
				}))
				DeferCleanup(server.Close)
				settings.ConnectURL = server.URL

				// Act
				actionErr := New(settings).Reconcile(context.Background(), &output)

				// Assert
				Expect(actionErr).NotTo(Succeed())
				Expect(requests.Load()).To(Equal(int32(1)))
			},
			Entry("[case] - unauthorized", http.StatusUnauthorized),
			Entry("[case] - forbidden", http.StatusForbidden),
			Entry("[case] - redirect", http.StatusTemporaryRedirect),
		)

		Context("[Behavior] - execution limits", func() {
			It("[case] - bounds retries by the total deadline", func(ctx SpecContext) {
				// Arrange
				server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					writer.WriteHeader(http.StatusServiceUnavailable)
				}))
				DeferCleanup(server.Close)
				settings.ConnectURL = server.URL
				settings.Timeout = 50 * time.Millisecond

				// Act
				err := New(settings).Reconcile(ctx, &output)

				// Assert
				Expect(err).To(MatchError(context.DeadlineExceeded))
			})
		})
	})

	Describe("[Method] - Connector.Validate", func() {
		var connector Connector

		BeforeEach(func() {
			connector = Connector{
				Name: "identity-outbox",
				Config: map[string]string{
					"connector.class":   "io.debezium.connector.postgresql.PostgresConnector",
					"database.hostname": "postgresql",
					"database.port":     "5432",
					"database.user":     "test",
					"database.password": "test-only",
					"database.dbname":   "test",
					"slot.name":         "outbox",
					"publication.name":  "outbox",
				},
			}
		})

		DescribeTable("[Behavior] - requires each database setting",
			func(key string) {
				// Arrange
				delete(connector.Config, key)

				// Act
				err := connector.Validate()

				// Assert
				Expect(err).To(MatchError("debezium.Connector.Validate: connector identity-outbox requires " + key))
			},
			Entry("[case] - connector class", "connector.class"),
			Entry("[case] - host", "database.hostname"),
			Entry("[case] - port", "database.port"),
			Entry("[case] - user", "database.user"),
			Entry("[case] - password", "database.password"),
			Entry("[case] - database", "database.dbname"),
			Entry("[case] - replication slot", "slot.name"),
			Entry("[case] - publication", "publication.name"),
		)

		DescribeTable("[Behavior] - rejects names that cannot be used in an endpoint path",
			func(name string) {
				// Arrange
				connector.Name = name

				// Act
				err := connector.Validate()

				// Assert
				Expect(err).To(MatchError("connector name must be a nonempty path segment"))
			},
			Entry("[case] - empty", ""),
			Entry("[case] - slash", "bad/name"),
			Entry("[case] - newline", "bad\nname"),
		)
	})
})
