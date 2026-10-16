//go:build integration

package integration

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var postgresqlServices = []string{"access-control", "identity", "notification", "organization", "hr", "template"}

func (fixture *clusterFixture) state(ctx context.Context) (platformState, error) {
	state := platformState{
		Volumes:     make(map[string]string),
		Credentials: make(map[string]string),
	}

	output, err := fixture.kubectl(ctx, "get", "namespace", "kube-system", "-o", "jsonpath={.metadata.uid}")
	if err != nil {
		return state, fmt.Errorf("integration.clusterFixture.state: read cluster identity: %w", err)
	}

	state.Namespace = output

	output, err = fixture.kubectl(ctx, "get", "pvc", "-A", "-o", "json")
	if err != nil {
		return state, fmt.Errorf("integration.clusterFixture.state: read persistent volumes: %w", err)
	}

	var volumes resourceList
	err = json.Unmarshal([]byte(output), &volumes)
	if err != nil {
		return state, err
	}

	for _, volume := range volumes.Items {
		key := volume.Metadata.Namespace + "/" + volume.Metadata.Name
		state.Volumes[key] = volume.Metadata.UID
	}

	state.Credentials, err = fixture.credentials(ctx)
	if err != nil {
		return state, err
	}

	material, err := os.ReadFile(filepath.Join(fixture.directory, "vault", "init.json"))
	if err != nil {
		return state, err
	}

	state.Material = fmt.Sprintf("%x", sha256.Sum256(material))

	return state, nil
}

func (fixture *clusterFixture) credentials(ctx context.Context) (map[string]string, error) {
	credentials := make(map[string]string)
	secrets := map[string][]string{
		"monadiam-apps": {"monadiam-bootstrap"},
		"monadiam-data": {
			"access-control-postgresql-app",
			"access-control-postgresql-debezium",
			"identity-postgresql-app",
			"identity-postgresql-debezium",
			"notification-postgresql-app",
			"notification-postgresql-debezium",
			"organization-postgresql-app",
			"organization-postgresql-debezium",
			"hr-postgresql-app",
			"hr-postgresql-debezium",
			"template-postgresql-app",
			"template-postgresql-debezium",
			"seaweedfs-main-s3",
			"seaweedfs-main-admin",
			"seaweedfs-main-archiver",
			"monadiam-data-bootstrap",
			"redis-main-auth",
			"clickhouse-main-app",
			"clickhouse-main-intercluster",
		},
		"monadiam-messaging": {
			"monadiam-messaging-bootstrap",
			"kafka-main-admin-password",
		},
		"monadiam-observability": {"grafana-admin"},
		"clickhouse-operator":    {"clickhouse-operator-altinity-clickhouse-operator"},
	}

	for namespace, names := range secrets {
		for _, name := range names {
			digest, err := fixture.credentialDigest(ctx, namespace, name)
			if err != nil {
				return nil, err
			}

			credentials[namespace+"/"+name] = digest
		}
	}

	return credentials, nil
}

func (fixture *clusterFixture) credentialDigest(ctx context.Context, namespace, name string) (string, error) {
	output, err := fixture.kubectl(ctx, "get", "secret", name, "-n", namespace, "-o", "json")
	if err != nil {
		return "", fmt.Errorf("integration.clusterFixture.credentialDigest: read secret %s/%s: %w", namespace, name, err)
	}

	var secret resource
	err = json.Unmarshal([]byte(output), &secret)
	if err != nil {
		return "", fmt.Errorf("integration.clusterFixture.credentialDigest: decode secret %s/%s", namespace, name)
	}

	if len(secret.Data) == 0 {
		return "", fmt.Errorf("integration.clusterFixture.credentialDigest: secret %s/%s is empty", namespace, name)
	}

	for key, value := range secret.Data {
		if len(value) == 0 {
			return "", fmt.Errorf("integration.clusterFixture.credentialDigest: secret %s/%s has an empty %s", namespace, name, key)
		}
	}

	data, err := json.Marshal(secret.Data)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

func (fixture *clusterFixture) vault(ctx context.Context) (vaultState, error) {
	var state vaultState
	output, err := fixture.cli(ctx, "vault", "status")
	if err != nil {
		return state, fmt.Errorf("integration.clusterFixture.vault: read Vault status: %w", err)
	}

	err = json.Unmarshal([]byte(output), &state)
	if err != nil {
		return state, err
	}

	file, err := os.Stat(filepath.Join(fixture.directory, "vault", "init.json"))
	if err != nil {
		return state, err
	}

	state.FileMode = file.Mode().Perm()

	directory, err := os.Stat(filepath.Join(fixture.directory, "vault"))
	if err != nil {
		return state, err
	}

	state.DirectoryMode = directory.Mode().Perm()

	return state, nil
}

func (fixture *clusterFixture) postgresqlValues(ctx context.Context) ([]string, error) {
	values := make([]string, 0, len(postgresqlServices))
	for _, service := range postgresqlServices {
		value, err := fixture.postgresqlDatabase(ctx, service+"-postgresql", service+"-service", "SELECT value FROM integration_probe WHERE id = 1")
		if err != nil {
			return nil, fmt.Errorf("read %s PostgreSQL: %w", service, err)
		}

		values = append(values, value)
	}

	return values, nil
}

func (fixture *clusterFixture) postgresqlDatabase(ctx context.Context, cluster, database, query string) (string, error) {
	primary, err := fixture.kubectl(ctx, "get", "cluster.postgresql.cnpg.io", cluster, "-n", "monadiam-data", "-o", "jsonpath={.status.currentPrimary}")
	if err != nil {
		return "", fmt.Errorf("integration.clusterFixture.postgresql: read PostgreSQL primary: %w", err)
	}

	output, err := fixture.kubectl(ctx, "exec", "-n", "monadiam-data", primary, "-c", "postgres", "--", "psql", "-U", "postgres", "-d", database, "-At", "-v", "ON_ERROR_STOP=1", "-c", query)

	return strings.TrimSpace(output), err
}

func (fixture *clusterFixture) seed(ctx context.Context) error {
	query := "CREATE TABLE IF NOT EXISTS integration_probe (id integer PRIMARY KEY, value text NOT NULL); " +
		"INSERT INTO integration_probe VALUES (1, '" + fixture.name + "') ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value;"
	for _, service := range postgresqlServices {
		_, err := fixture.postgresqlDatabase(ctx, service+"-postgresql", service+"-service", query)
		if err != nil {
			return err
		}
	}

	return nil
}
