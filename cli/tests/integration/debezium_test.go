//go:build integration

package integration

import (
	"fmt"
	"io"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/debezium"
)

var _ = Describe("[Infrastructure] - Debezium outbox", Ordered, ContinueOnFailure, Label("integration"), func() {
	var fixture composeFixture
	var settings config.Debezium
	var client *connectClient

	BeforeAll(func(ctx SpecContext) {
		fixture.project = fmt.Sprintf("infra-cli-test-%d", time.Now().UnixNano())
		DeferCleanup(func(ctx SpecContext) {
			if CurrentSpecReport().Failed() {
				logs, _ := fixture.compose(ctx, "logs", "--tail", "40")
				_, _ = io.WriteString(GinkgoWriter, logs)
			}

			output, err := fixture.compose(ctx, "down", "--volumes", "--remove-orphans")
			Expect(err).NotTo(HaveOccurred(), output)
		}, NodeTimeout(time.Minute))

		output, err := fixture.compose(ctx, "up", "-d", "--build", "--wait", "--wait-timeout", "180")
		Expect(err).NotTo(HaveOccurred(), output)

		address, err := fixture.compose(ctx, "port", "connect", "8083")
		Expect(err).NotTo(HaveOccurred())
		settings = integrationSettings("http://" + strings.TrimSpace(address))
		client = newConnectClient(settings.ConnectURL)
	}, NodeTimeout(8*time.Minute))

	Describe("[Behavior] - repeated connector registration", func() {
		It("[case] - keeps one running connector under the same name", func(ctx SpecContext) {
			// Arrange
			service := debezium.New(settings)

			// Act
			firstErr := service.Reconcile(ctx, GinkgoWriter)
			secondErr := service.Reconcile(ctx, GinkgoWriter)
			readyErr := client.waitRunning(ctx, "test-service-outbox-connector")
			var names []string
			namesErr := client.readJSON(ctx, "/connectors", &names)

			// Assert
			Expect(firstErr).To(Succeed())
			Expect(secondErr).To(Succeed())
			Expect(readyErr).To(Succeed())
			Expect(namesErr).To(Succeed())
			Expect(names).To(ConsistOf("test-service-outbox-connector"))
		}, SpecTimeout(3*time.Minute))
	})

	Describe("[Behavior] - outbox delivery", func() {
		It("[case] - delivers a PostgreSQL outbox event to its Kafka topic", func(ctx SpecContext) {
			// Arrange
			Expect(debezium.New(settings).Reconcile(ctx, GinkgoWriter)).To(Succeed())
			Expect(client.waitRunning(ctx, "test-service-outbox-connector")).To(Succeed())

			output, err := fixture.compose(ctx, "exec", "-T", "kafka", "/opt/kafka/bin/kafka-topics.sh", "--bootstrap-server", "kafka:29092", "--create", "--if-not-exists", "--topic", "realm", "--partitions", "4", "--replication-factor", "1")
			Expect(err).NotTo(HaveOccurred(), output)

			// Act
			message, deliveryErr := fixture.deliverOutbox(ctx)

			// Assert
			Expect(deliveryErr).NotTo(HaveOccurred(), message)
			Expect(message).To(ContainSubstring("outbox-delivered"))
		}, SpecTimeout(3*time.Minute))
	})
})
