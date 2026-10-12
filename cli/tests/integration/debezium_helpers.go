//go:build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/MonadIAM/infrastructure/cli/config"
)

type connectClient struct {
	http *resty.Client
}

type connectorStatus struct {
	Connector taskStatus   `json:"connector"`
	Tasks     []taskStatus `json:"tasks"`
}

func newConnectClient(endpoint string) *connectClient {
	return &connectClient{http: resty.New().SetBaseURL(endpoint).SetTimeout(5 * time.Second)}
}

func (client *connectClient) waitRunning(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		var status connectorStatus
		err := client.readJSON(ctx, "/connectors/"+name+"/status", &status)
		if err == nil && status.running() {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("integration.connectClient.waitRunning: last status %+v: %w", status, errors.Join(ctx.Err(), err))
		case <-ticker.C:
		}
	}
}

func (client *connectClient) readJSON(ctx context.Context, path string, destination any) error {
	response, err := client.http.R().SetContext(ctx).SetResult(destination).Get(path)
	if err != nil {
		return fmt.Errorf("integration.connectClient.readJSON: %w", err)
	}

	if response.StatusCode() != http.StatusOK {
		return fmt.Errorf("integration.connectClient.readJSON: connect returned HTTP %d", response.StatusCode())
	} else {
		return nil
	}
}

func (status connectorStatus) running() bool {
	return status.Connector.State == "RUNNING" &&
		len(status.Tasks) == 1 &&
		status.Tasks[0].State == "RUNNING"
}

func (fixture composeFixture) deliverOutbox(ctx context.Context) (string, error) {
	query := `INSERT INTO system.outbox (id, destination_topic, action_type, payload) VALUES ('00000000-0000-4000-8000-000000000001', 'realm', 'create', '{"realm":"00000000-0000-4000-8000-000000000002","proof":"outbox-delivered"}')`
	output, err := fixture.compose(ctx, "exec", "-T", "postgresql", "psql", "-U", "test", "-d", "test", "-c", query)
	if err != nil {
		return output, err
	}

	return fixture.compose(ctx, "exec", "-T", "kafka", "/opt/kafka/bin/kafka-console-consumer.sh", "--bootstrap-server", "kafka:29092", "--topic", "realm", "--from-beginning", "--max-messages", "1", "--timeout-ms", "30000")
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
