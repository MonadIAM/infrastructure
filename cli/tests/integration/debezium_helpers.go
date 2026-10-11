//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/MonadIAM/infrastructure/cli/config"
)

func waitRunningConnector(ctx context.Context, endpoint string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		var status connectorStatus
		err := readConnectJSON(ctx, endpoint, &status)
		if err == nil && status.running() {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for running connector: %w (last status: %+v; last error: %v)", ctx.Err(), status, err)
		case <-ticker.C:
		}
	}
}

func readConnectJSON(ctx context.Context, endpoint string, destination any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	client := http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return err
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("connect returned HTTP %d", response.StatusCode)
	}

	return json.NewDecoder(response.Body).Decode(destination)
}

func integrationSettings(endpoint string) config.Debezium {
	return config.Debezium{
		ConnectURL:                  endpoint,
		TemplatePath:                "../../../messaging/debezium/outbox.connector.json",
		Timeout:                     time.Minute,
		RequestTimeout:              10 * time.Second,
		RetryInterval:               time.Second,
		RealmPartitions:             4,
		PositionPartitions:          2,
		ProjectionJournalPartitions: 8,
		Databases: []config.Database{
			{
				Service:     "test-service",
				Host:        "postgresql",
				Name:        "test",
				User:        "test",
				Password:    "test-only",
				Slot:        "test_outbox",
				Publication: "test_outbox",
			},
		},
	}
}

func (fixture composeFixture) deliverOutbox(ctx context.Context) (string, error) {
	query := `INSERT INTO system.outbox (id, destination_topic, action_type, payload) VALUES ('00000000-0000-4000-8000-000000000001', 'realm', 'create', '{"realm":"00000000-0000-4000-8000-000000000002","proof":"outbox-delivered"}')`
	output, err := fixture.compose(ctx, "exec", "-T", "postgresql", "psql", "-U", "test", "-d", "test", "-c", query)
	if err != nil {
		return output, err
	}

	return fixture.compose(ctx, "exec", "-T", "kafka", "/opt/kafka/bin/kafka-console-consumer.sh", "--bootstrap-server", "kafka:29092", "--topic", "realm", "--from-beginning", "--max-messages", "1", "--timeout-ms", "30000")
}
