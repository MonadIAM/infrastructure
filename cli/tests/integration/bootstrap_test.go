//go:build integration

package integration

import (
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/MonadIAM/infrastructure/cli/platform/clickhouse"
)

var _ = Describe("[Infrastructure] - Local bootstrap", Ordered, ContinueOnFailure, Label("integration"), func() {
	var fixture composeFixture

	BeforeAll(func(ctx SpecContext) {
		fixture = composeFixture{
			project: fmt.Sprintf("infra-cli-test-%d", time.Now().UnixNano()),
			file:    "testdata/bootstrap.yml",
		}
		DeferCleanup(func(ctx SpecContext) {
			if CurrentSpecReport().Failed() {
				logs, _ := fixture.compose(ctx, "logs", "--tail", "30")
				_, _ = io.WriteString(GinkgoWriter, logs)
			}

			output, err := fixture.compose(ctx, "--profile", "tools", "down", "--volumes", "--remove-orphans")
			Expect(err).NotTo(HaveOccurred(), output)
		}, NodeTimeout(time.Minute))

		output, err := fixture.compose(ctx, "build", "cli")
		Expect(err).NotTo(HaveOccurred(), output)

		output, err = fixture.compose(ctx, "up", "-d", "--build", "--wait", "--wait-timeout", "240")
		Expect(err).NotTo(HaveOccurred(), output)
		Expect(fixture.initialize(ctx)).To(Succeed())
	}, NodeTimeout(8*time.Minute))

	Describe("[Behavior] - repeated initialization", func() {
		It("[case] - preserves Vault keys and token identity while reapplying resource settings", func(ctx SpecContext) {
			// Arrange
			originalKey, err := fixture.vaultField(ctx, "transit/keys/identity-access-token", "keys")
			Expect(err).NotTo(HaveOccurred())
			originalToken, err := fixture.vaultField(ctx, "auth/token/lookup-self", "accessor")
			Expect(err).NotTo(HaveOccurred())

			// Act
			actionErr := fixture.initialize(ctx)
			key, keyErr := fixture.vaultField(ctx, "transit/keys/identity-access-token", "keys")
			token, tokenErr := fixture.vaultField(ctx, "auth/token/lookup-self", "accessor")
			policies, policiesErr := fixture.vaultField(ctx, "auth/approle/role/notification-service", "token_policies")

			// Assert
			Expect(actionErr).To(Succeed())
			Expect(keyErr).NotTo(HaveOccurred())
			Expect(tokenErr).NotTo(HaveOccurred())
			Expect(policiesErr).NotTo(HaveOccurred())
			Expect(key).To(Equal(originalKey))
			Expect(token).To(Equal(originalToken))
			Expect(policies).To(ContainSubstring("notification-service-transit"))
		}, SpecTimeout(3*time.Minute))
	})

	Describe("[Behavior] - parameterized queries", func() {
		It("[case] - preserves string bytes and binds table identifiers on the server", func(ctx SpecContext) {
			// Arrange
			address, err := fixture.compose(ctx, "port", "clickhouse", "8123")
			Expect(err).NotTo(HaveOccurred())
			client := clickhouse.New(clickhouse.Settings{
				Address:  "http://" + strings.TrimSpace(address),
				User:     "default",
				Password: "test-only",
				Timeout:  time.Second * 10,
			}, false)
			value := "quote'\"; SELECT 1; --\t\n\r\x00\\N\\t"
			parameters := map[string]string{
				"value":    value,
				"database": "system",
				"table":    "one",
			}
			query := "SELECT hex({value:String}) FROM {database:Identifier}.{table:Identifier}"

			// Act
			result, err := client.Query(ctx, query, parameters)

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(strings.ToUpper(hex.EncodeToString([]byte(value)))))
		}, SpecTimeout(time.Minute))
	})

	Describe("[Behavior] - ordered topic protection", func() {
		It("[case] - refuses to change the existing realm partition count", func(ctx SpecContext) {
			// Arrange
			args := []string{
				"run", "--rm", "--no-deps", "-e", "REALM_PARTITIONS=5",
				"cli", "kafka", "reconcile", "--env-file=",
			}

			// Act
			output, err := fixture.compose(ctx, args...)
			metadata, metadataErr := fixture.compose(ctx, "exec", "-T", "kafka", "/opt/kafka/bin/kafka-topics.sh", "--bootstrap-server", "kafka:29092", "--describe", "--topic", "realm")

			// Assert
			Expect(err).To(HaveOccurred())
			Expect(output).To(ContainSubstring("realm has 4 partitions; expected 5"))
			Expect(metadataErr).NotTo(HaveOccurred(), metadata)
			Expect(metadata).To(MatchRegexp(`PartitionCount:\s*4\b`))
		}, SpecTimeout(time.Minute))
	})

	Describe("[Behavior] - archive permissions", func() {
		It("[case] - allows the archiver to write objects but rejects deletion", func(ctx SpecContext) {
			// Arrange
			client, settings := fixture.storage(ctx)
			bucket := settings.Buckets[0].Name

			// Act
			_, writeErr := client.PutObject(ctx, bucket, "permission-test", strings.NewReader("test"), 4, minio.PutObjectOptions{})
			deleteErr := client.RemoveObject(ctx, bucket, "permission-test", minio.RemoveObjectOptions{})
			contents, readErr := readObject(ctx, client, bucket, "permission-test")

			// Assert
			Expect(writeErr).NotTo(HaveOccurred())
			Expect(minio.ToErrorResponse(deleteErr).Code).To(Equal("AccessDenied"))
			Expect(readErr).NotTo(HaveOccurred())
			Expect(contents).To(Equal("test"))
		}, SpecTimeout(time.Minute))
	})

	Describe("[Behavior] - partition archival", func() {
		It("[case] - exports both tables and records each partition once across repeated cycles", func(ctx SpecContext) {
			// Arrange
			createdAt := time.Now().UTC().AddDate(0, 0, -160)
			partition := createdAt.Format("200601")
			Expect(fixture.seedArchive(ctx, createdAt)).To(Succeed())
			client, settings := fixture.storage(ctx)

			expectedIDs := map[string]string{
				"audit-archive":      "00000000-0000-4000-8000-000000000001",
				"change-log-archive": "00000000-0000-4000-8000-000000000002",
			}

			// Act
			firstOutput, firstErr := fixture.cli(ctx, "archive", "run")
			secondOutput, secondErr := fixture.cli(ctx, "archive", "run")
			rows, rowsErr := fixture.clickhouse(ctx, "SELECT table_name, partition, count(), sum(row_count) FROM audit.archive_log GROUP BY table_name, partition ORDER BY table_name")
			objects, objectsErr := fixture.readArchive(ctx, client, settings, partition)

			// Assert
			Expect(firstErr).NotTo(HaveOccurred(), firstOutput)
			Expect(secondErr).NotTo(HaveOccurred(), secondOutput)
			Expect(rowsErr).NotTo(HaveOccurred(), rows)
			Expect(objectsErr).NotTo(HaveOccurred())
			Expect(strings.TrimSpace(rows)).To(Equal("audit_log\t" + partition + "\t1\t1\nchange_log\t" + partition + "\t1\t1"))
			Expect(objects).To(HaveLen(len(expectedIDs)))

			for bucket, id := range expectedIDs {
				Expect(objects).To(HaveKey(bucket))
				Expect(objects[bucket].size).To(BeNumerically(">", 0), bucket)
				Expect(objects[bucket].rows).To(Equal(id+"\tintegration"), bucket)
			}
		}, SpecTimeout(3*time.Minute))
	})
})
