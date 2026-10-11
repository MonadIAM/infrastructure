package debezium

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MonadIAM/infrastructure/cli/config"
)

var placeholder = regexp.MustCompile(`\$\{([^{}]+)\}`)

func LoadConnectors(settings config.Debezium) ([]Connector, error) {
	err := settings.Validate()
	if err != nil {
		return nil, err
	}

	template, err := os.ReadFile(settings.TemplatePath)
	if err == nil {
		return Build(settings, template)
	} else {
		return nil, fmt.Errorf("cannot read connector template %s", settings.TemplatePath)
	}
}

func Build(settings config.Debezium, template []byte) ([]Connector, error) {
	err := settings.Validate()
	if err != nil {
		return nil, err
	}

	var fields map[string]string
	err = json.Unmarshal(template, &fields)
	if err != nil || len(fields) == 0 {
		return nil, errors.New("connector template must be a nonempty JSON object with string values")
	}

	connectors := make([]Connector, 0, len(settings.Databases))

	for _, database := range settings.Databases {
		values := templateValues(settings, database)
		connector, err := renderConnector(database.Service, fields, values)
		if err != nil {
			return nil, err
		}

		connectors = append(connectors, connector)
	}

	return connectors, nil
}

func templateValues(settings config.Debezium, database config.Database) map[string]string {
	return map[string]string{
		"SERVICE_NAME":                  database.Service,
		"POSTGRES_ROUTE_HOST":           database.Host,
		"POSTGRES_ROUTE_PORT":           "5432",
		"POSTGRES_DB":                   database.Name,
		"POSTGRES_USER":                 database.User,
		"POSTGRES_PASSWORD":             database.Password,
		"POSTGRES_OUTBOX_SLOT":          database.Slot,
		"POSTGRES_OUTBOX_PUBLICATION":   database.Publication,
		"REALM_PARTITIONS":              strconv.Itoa(settings.RealmPartitions),
		"POSITION_PARTITIONS":           strconv.Itoa(settings.PositionPartitions),
		"PROJECTION_JOURNAL_PARTITIONS": strconv.Itoa(settings.ProjectionJournalPartitions),
		"routedByValue":                 "${routedByValue}",
	}
}

func renderConnector(service string, fields map[string]string, values map[string]string) (Connector, error) {
	connector := Connector{
		Name:   service + "-outbox-connector",
		Config: make(map[string]string, len(fields)),
	}

	for key, value := range fields {
		for _, token := range placeholder.FindAllString(value, -1) {
			_, exists := values[token[2:len(token)-1]]
			if !exists {
				return Connector{}, errors.New("connector template contains an unsupported placeholder")
			}
		}

		connector.Config[key] = placeholder.ReplaceAllStringFunc(value, func(token string) string {
			return values[token[2:len(token)-1]]
		})
	}

	return connector, connector.Validate()
}

func Reconcile(ctx context.Context, settings config.Debezium, output io.Writer) error {
	connectors, err := LoadConnectors(settings)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, settings.Timeout)
	defer cancel()

	client := connectClient{
		http: &http.Client{
			Timeout: settings.RequestTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		endpoint:      strings.TrimRight(settings.ConnectURL, "/"),
		retryInterval: settings.RetryInterval,
	}

	_, err = fmt.Fprintln(output, "Waiting for Kafka Connect...")
	if err != nil {
		return err
	}

	err = client.waitUntilReady(ctx)
	if err != nil {
		return err
	}

	for _, connector := range connectors {
		err := client.register(ctx, connector)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(output, "Registered connector %s.\n", connector.Name)
		if err != nil {
			return err
		}
	}

	_, err = fmt.Fprintf(output, "Registered %d outbox connectors.\n", len(connectors))
	return err
}

func (client connectClient) waitUntilReady(ctx context.Context) error {
	err := retry(ctx, client.retryInterval, func() error {
		return client.request(ctx, http.MethodGet, "/", nil)
	})
	if err == nil {
		return nil
	} else {
		return fmt.Errorf("waiting for Kafka Connect: %w", err)
	}
}

func (client connectClient) register(ctx context.Context, connector Connector) error {
	body, err := json.Marshal(connector.Config)
	if err != nil {
		return fmt.Errorf("cannot encode connector %s", connector.Name)
	}

	path := "/connectors/" + connector.Name + "/config"
	err = retry(ctx, client.retryInterval, func() error {
		return client.request(ctx, http.MethodPut, path, body)
	})
	if err == nil {
		return nil
	} else {
		return fmt.Errorf("registering connector %s: %w", connector.Name, err)
	}
}

func (client connectClient) request(ctx context.Context, method string, path string, body []byte) error {
	request, err := http.NewRequestWithContext(ctx, method, client.endpoint+path, bytes.NewReader(body))
	if err != nil {
		return errors.New("cannot create Kafka Connect request")
	}

	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := client.http.Do(request)
	if err != nil {
		contextErr := ctx.Err()
		if contextErr == nil {
			return requestError{temporary: true}
		} else {
			return contextErr
		}
	}

	defer func() { _ = response.Body.Close() }()

	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	} else {
		return requestError{
			status:    response.StatusCode,
			temporary: response.StatusCode == 409 || response.StatusCode == 429 || response.StatusCode >= 500,
		}
	}
}

func retry(ctx context.Context, interval time.Duration, action func() error) error {
	for {
		err := ctx.Err()
		if err != nil {
			return err
		}

		err = action()
		if err == nil {
			return nil
		}

		var failure requestError
		if !errors.As(err, &failure) || !failure.temporary {
			return err
		}

		timer := time.NewTimer(interval)

		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("%w (last error: %v)", ctx.Err(), err)
		case <-timer.C:
		}
	}
}
