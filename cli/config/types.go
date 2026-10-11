package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	infrahttp "github.com/MonadIAM/infrastructure/cli/platform/http"
)

type Database struct {
	Service     string
	Host        string
	Name        string
	User        string
	Password    string
	Slot        string
	Publication string
}

type Debezium struct {
	ConnectURL                  string
	TemplatePath                string
	Timeout                     time.Duration
	RequestTimeout              time.Duration
	RetryInterval               time.Duration
	RealmPartitions             int
	PositionPartitions          int
	ProjectionJournalPartitions int
	Databases                   []Database
}

func (database Database) Validate() error {
	if database.Service == "" || strings.ContainsAny(database.Service, "/?#% \t\r\n") {
		return errors.New("outbox service name must be a nonempty path segment")
	}

	fields := map[string]string{
		"host":        database.Host,
		"database":    database.Name,
		"user":        database.User,
		"password":    database.Password,
		"slot":        database.Slot,
		"publication": database.Publication,
	}

	requiredFields := []string{"host", "database", "user", "password", "slot", "publication"}

	for _, field := range requiredFields {
		if fields[field] == "" {
			return fmt.Errorf("%s: outbox %s is required", database.Service, field)
		}
	}

	return nil
}

func (settings Debezium) Validate() error {
	err := infrahttp.ValidateURL(settings.ConnectURL)
	if err != nil {
		return fmt.Errorf("CONNECT_URL: %w", err)
	}

	if settings.TemplatePath == "" {
		return errors.New("CONNECTOR_TEMPLATE is required")
	}

	if settings.Timeout <= 0 || settings.RequestTimeout <= 0 || settings.RetryInterval <= 0 {
		return errors.New("Debezium timeouts and retry interval must be positive")
	}

	if settings.RealmPartitions < 1 || settings.PositionPartitions < 1 || settings.ProjectionJournalPartitions < 1 {
		return errors.New("Debezium partition counts must be positive")
	}

	if len(settings.Databases) == 0 {
		return errors.New("at least one outbox database is required")
	}

	services := make(map[string]bool)

	for _, database := range settings.Databases {
		err := database.Validate()
		if err != nil {
			return err
		}

		if services[database.Service] {
			return fmt.Errorf("duplicate outbox service: %s", database.Service)
		}

		services[database.Service] = true
	}

	return nil
}

type Environment map[string]string
