package debezium

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"

	"github.com/go-resty/resty/v2"

	"github.com/MonadIAM/infrastructure/cli/config"
	infrahttp "github.com/MonadIAM/infrastructure/cli/platform/http"
)

type Service struct {
	settings config.Debezium
	client   *resty.Client
}

var placeholder = regexp.MustCompile(`\$\{([^{}]+)\}`)

func New(settings config.Debezium) *Service {
	return &Service{settings: settings}
}

func (service *Service) LoadConnectors() ([]Connector, error) {
	err := service.settings.Validate()
	if err != nil {
		return nil, err
	}

	template, err := os.ReadFile(service.settings.TemplatePath)
	if err == nil {
		return service.Build(template)
	} else {
		return nil, fmt.Errorf("debezium.Service.LoadConnectors: read connector template %s: %w", service.settings.TemplatePath, err)
	}
}

func (service *Service) Build(template []byte) ([]Connector, error) {
	err := service.settings.Validate()
	if err != nil {
		return nil, err
	}

	var fields map[string]string
	err = json.Unmarshal(template, &fields)
	if err != nil || len(fields) == 0 {
		return nil, errors.New("connector template must be a nonempty JSON object with string values")
	}

	connectors := make([]Connector, 0, len(service.settings.Databases))

	for _, database := range service.settings.Databases {
		connector, err := service.renderConnector(database, fields)
		if err != nil {
			return nil, err
		}

		connectors = append(connectors, connector)
	}

	return connectors, nil
}

func (service *Service) Reconcile(ctx context.Context, output io.Writer) error {
	connectors, err := service.LoadConnectors()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, service.settings.Timeout)
	defer cancel()

	service.client = infrahttp.New(service.settings.ConnectURL).
		SetTimeout(service.settings.RequestTimeout).
		SetRetryCount(int(service.settings.Timeout/service.settings.RetryInterval) + 1).
		SetRetryWaitTime(service.settings.RetryInterval).
		SetRetryMaxWaitTime(service.settings.RetryInterval).
		AddRetryCondition(func(response *resty.Response, err error) bool {
			if response != nil && response.StatusCode() != 0 {
				code := response.StatusCode()
				return code == 409 || code == 429 || code >= 500
			} else {
				return err != nil
			}
		})

	_, err = fmt.Fprintln(output, "Waiting for Kafka Connect...")
	if err != nil {
		return err
	}

	err = service.waitUntilReady(ctx)
	if err != nil {
		return err
	}

	for _, connector := range connectors {
		err := service.register(ctx, connector)
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

func (service *Service) waitUntilReady(ctx context.Context) error {
	response, err := service.client.R().SetContext(ctx).Get("/")
	if err != nil {
		return fmt.Errorf("debezium.Service.waitUntilReady: %w", infrahttp.RequestError{Cause: err})
	}

	if response.IsSuccess() {
		return nil
	} else {
		return fmt.Errorf("debezium.Service.waitUntilReady: HTTP status %d", response.StatusCode())
	}
}

func (service *Service) renderConnector(database config.Database, fields map[string]string) (Connector, error) {
	values := map[string]string{
		"SERVICE_NAME":                  database.Service,
		"POSTGRES_ROUTE_HOST":           database.Host,
		"POSTGRES_ROUTE_PORT":           "5432",
		"POSTGRES_DB":                   database.Name,
		"POSTGRES_USER":                 database.User,
		"POSTGRES_PASSWORD":             database.Password,
		"POSTGRES_OUTBOX_SLOT":          database.Slot,
		"POSTGRES_OUTBOX_PUBLICATION":   database.Publication,
		"REALM_PARTITIONS":              strconv.Itoa(service.settings.RealmPartitions),
		"POSITION_PARTITIONS":           strconv.Itoa(service.settings.PositionPartitions),
		"PROJECTION_JOURNAL_PARTITIONS": strconv.Itoa(service.settings.ProjectionJournalPartitions),
		"routedByValue":                 "${routedByValue}",
	}

	connector := Connector{
		Name:   database.Service + "-outbox-connector",
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

func (service *Service) register(ctx context.Context, connector Connector) error {
	response, err := service.client.R().
		SetContext(ctx).
		SetBody(connector.Config).
		Put("/connectors/" + connector.Name + "/config")
	if err != nil {
		return fmt.Errorf("debezium.Service.register: connector %s: %w", connector.Name, infrahttp.RequestError{Cause: err})
	}

	if response.IsSuccess() {
		return nil
	} else {
		return fmt.Errorf("debezium.Service.register: connector %s: HTTP status %d", connector.Name, response.StatusCode())
	}
}
