package config

import (
	"errors"
	"fmt"
	"strings"
	"time"
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
	ConnectURL                  string        `koanf:"CONNECT_URL"`
	TemplatePath                string        `koanf:"CONNECTOR_TEMPLATE"`
	Timeout                     time.Duration `koanf:"DEBEZIUM_TIMEOUT"`
	RequestTimeout              time.Duration `koanf:"DEBEZIUM_REQUEST_TIMEOUT"`
	RetryInterval               time.Duration `koanf:"DEBEZIUM_RETRY_INTERVAL"`
	RealmPartitions             int           `koanf:"REALM_PARTITIONS"`
	PositionPartitions          int           `koanf:"POSITION_PARTITIONS"`
	ProjectionJournalPartitions int           `koanf:"PROJECTION_JOURNAL_PARTITIONS"`
	Databases                   []Database    `koanf:"-"`
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
			return fmt.Errorf("config.Database.Validate: %s: outbox %s is required", database.Service, field)
		}
	}

	return nil
}

func (settings Debezium) Validate() error {
	if settings.ConnectURL == "" {
		return errors.New("CONNECT_URL is required")
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
			return fmt.Errorf("config.Debezium.Validate: duplicate outbox service: %s", database.Service)
		}

		services[database.Service] = true
	}

	return nil
}
