//go:build integration

package integration

import (
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[Infrastructure] - Kafka", Label("integration", "kafka"), func() {
	for _, service := range services {
		Describe("[Service] - "+service, func() {
			Describe("[Behavior] - CDC initialization", func() {
				It("[case] - starts the connector after the application creates its outbox", func(ctx SpecContext) {
					// Arrange
					client := fixture.postgresql(ctx, service, "service/"+service+"-postgresql-rw")
					connection, err := client.connect(ctx)
					Expect(err).NotTo(HaveOccurred())
					schema, err := os.ReadFile("testdata/kafka/outbox.sql")
					Expect(err).NotTo(HaveOccurred())
					connector := "kafkaconnector/" + service + "-postgresql-outbox"
					publication := strings.ReplaceAll(service, "-", "_") + "_postgresql_outbox_publication"

					// Act
					_, schemaErr := connection.Exec(ctx, string(schema))
					output, restartErr := fixture.kubectl(ctx, "annotate", connector, "-n", "monadiam-messaging",
						"strimzi.io/restart=includeTasks,onlyFailed", "--overwrite")

					// Assert
					Expect(schemaErr).NotTo(HaveOccurred())
					Expect(restartErr).NotTo(HaveOccurred(), output)
					output, err = fixture.kubectl(ctx, "wait", connector, "-n", "monadiam-messaging",
						"--for=condition=Ready", "--timeout=300s")
					Expect(err).NotTo(HaveOccurred(), output)
					Eventually(func() (string, error) {
						return fixture.kubectl(ctx, "get", connector, "-n", "monadiam-messaging",
							"-o", "jsonpath={.status.connectorStatus.connector.state}/{.status.connectorStatus.tasks[0].state}")
					}).WithContext(ctx).WithTimeout(3 * time.Minute).WithPolling(time.Second).Should(Equal("RUNNING/RUNNING"))
					var published bool
					err = connection.QueryRow(ctx, "SELECT EXISTS (SELECT FROM pg_publication_tables WHERE pubname = $1 AND schemaname = 'system' AND tablename = 'outbox')", publication).Scan(&published)
					Expect(err).NotTo(HaveOccurred())
					Expect(published).To(BeTrue())
				}, NodeTimeout(9*time.Minute))
			})
		})
	}
})
