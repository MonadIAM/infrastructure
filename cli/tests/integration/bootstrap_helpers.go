//go:build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/MonadIAM/infrastructure/cli/platform/clickhouse"
	"github.com/MonadIAM/infrastructure/cli/platform/seaweedfs"
)

func (fixture composeFixture) initialize(ctx context.Context) error {
	commands := [][]string{
		{"kafka", "reconcile"},
		{"vault", "initialize"},
		{"vault", "bootstrap"},
		{"seaweedfs", "reconcile"},
		{"clickhouse", "reconcile"},
	}

	for _, command := range commands {
		output, err := fixture.cli(ctx, command...)
		if err != nil {
			return fmt.Errorf("%s: %w\n%s", strings.Join(command, " "), err, output)
		}
	}

	return nil
}

func (fixture composeFixture) vaultField(ctx context.Context, path, field string) (string, error) {
	output, err := fixture.compose(ctx, "exec", "-T", "-e", "VAULT_ADDR=http://127.0.0.1:8200", "-e", "VAULT_TOKEN=integration-local-token", "vault", "vault", "read", "-field="+field, path)

	return strings.TrimSpace(output), err
}

func (fixture composeFixture) clickhouse(ctx context.Context, query string) (string, error) {
	return fixture.compose(ctx, "exec", "-T", "clickhouse", "clickhouse-client", "--password", "test-only", "--query", query)
}

func (fixture composeFixture) storage(ctx context.Context) (*minio.Client, seaweedfs.Settings) {
	ginkgo.GinkgoHelper()
	environment, err := godotenv.Read("../../../.env.example")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	address, err := fixture.compose(ctx, "port", "seaweedfs", "8333")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	environment["SEAWEEDFS_ENDPOINT"] = "http://" + strings.TrimSpace(address)

	settings, err := seaweedfs.Load(environment)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	client, err := seaweedfs.Client(settings, false)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return client, settings
}

func (fixture composeFixture) seedArchive(ctx context.Context, createdAt time.Time) error {
	timestamp := createdAt.Format("2006-01-02 15:04:05")
	queries := []string{
		"INSERT INTO audit.audit_log (id, service, created_at, action_type, entity_type) VALUES ('00000000-0000-4000-8000-000000000001', 'integration', '" + timestamp + "', 'create', 'test')",
		"INSERT INTO audit.change_log (id, service, created_at, audit_entry, change_type, entity_type, entity) VALUES ('00000000-0000-4000-8000-000000000002', 'integration', '" + timestamp + "', generateUUIDv4(), 'create', 'test', generateUUIDv4())",
	}

	for _, query := range queries {
		output, err := fixture.clickhouse(ctx, query)
		if err != nil {
			return fmt.Errorf("seed archive: %w\n%s", err, output)
		}
	}

	return nil
}

func readObject(ctx context.Context, client *minio.Client, bucket, key string) (string, error) {
	object, err := client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return "", err
	}

	data, readErr := io.ReadAll(object)
	closeErr := object.Close()

	return string(data), errors.Join(readErr, closeErr)
}

func (fixture composeFixture) readArchive(ctx context.Context, client *minio.Client, settings seaweedfs.Settings, partition string) (map[string]archiveObject, error) {
	objects := make(map[string]archiveObject)
	key := partition + ".parquet"

	for _, bucket := range settings.Buckets {
		object, err := client.StatObject(ctx, bucket.Name, key, minio.StatObjectOptions{})
		if err != nil {
			return nil, fmt.Errorf("inspect archive %s: %w", bucket.Name, err)
		}

		address := "http://seaweedfs:8333/" + bucket.Name + "/" + key
		query := "SELECT toString(id), service FROM s3(" + clickhouse.Quote(address) + ", " + clickhouse.Quote(settings.ArchiverKey) + ", " + clickhouse.Quote(settings.ArchiverSecret) + ", 'Parquet')"
		rows, err := fixture.clickhouse(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("read archive %s: %w", bucket.Name, err)
		}

		objects[bucket.Name] = archiveObject{size: object.Size, rows: strings.TrimSpace(rows)}
	}

	return objects, nil
}
