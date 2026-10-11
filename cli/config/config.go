package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func Load(path string, overrides map[string]string) (Environment, error) {
	values := make(Environment)

	if path != "" {
		var err error
		values, err = godotenv.Read(path)
		if err != nil {
			return nil, fmt.Errorf("cannot read or parse environment file %s", path)
		}
	}

	for _, entry := range os.Environ() {
		key, value, _ := strings.Cut(entry, "=")
		values[key] = value
	}

	for key, value := range overrides {
		values[key] = value
	}

	return values, nil
}

func ParseDebezium(values Environment) (Debezium, error) {
	settings := Debezium{
		ConnectURL:   values.Get("CONNECT_URL", "http://kafka-connect:8083"),
		TemplatePath: values.Get("CONNECTOR_TEMPLATE", "messaging/debezium/outbox.connector.json"),
	}

	var err error

	settings.RealmPartitions, err = values.PositiveInt("REALM_PARTITIONS")
	if err != nil {
		return Debezium{}, err
	}

	settings.PositionPartitions, err = values.PositiveInt("POSITION_PARTITIONS")
	if err != nil {
		return Debezium{}, err
	}

	settings.ProjectionJournalPartitions, err = values.PositiveInt("PROJECTION_JOURNAL_PARTITIONS")
	if err != nil {
		return Debezium{}, err
	}

	settings.Timeout, err = values.Duration("DEBEZIUM_TIMEOUT", "2m")
	if err != nil {
		return Debezium{}, err
	}

	settings.RequestTimeout, err = values.Duration("DEBEZIUM_REQUEST_TIMEOUT", "10s")
	if err != nil {
		return Debezium{}, err
	}

	settings.RetryInterval, err = values.Duration("DEBEZIUM_RETRY_INTERVAL", "2s")
	if err != nil {
		return Debezium{}, err
	}

	for _, service := range Services() {
		settings.Databases = append(settings.Databases, readDatabase(values, service))
	}

	return settings, settings.Validate()
}

func readDatabase(values map[string]string, service string) Database {
	prefix := Prefix(service) + "_POSTGRES_"

	return Database{
		Service:     service + "-service",
		Host:        service + "-postgresql",
		Name:        values[prefix+"DB"],
		User:        values[prefix+"USER"],
		Password:    values[prefix+"PASSWORD"],
		Slot:        values[prefix+"OUTBOX_SLOT"],
		Publication: values[prefix+"OUTBOX_PUBLICATION"],
	}
}

func (environment Environment) Get(key, fallback string) string {
	value, exists := environment[key]
	if exists {
		return value
	} else {
		return fallback
	}
}

func (environment Environment) Require(keys ...string) error {
	for _, key := range keys {
		if environment[key] == "" {
			return fmt.Errorf("%s is required", key)
		}
	}

	return nil
}

func (environment Environment) PositiveInt(key string) (int, error) {
	value, err := strconv.Atoi(environment[key])
	if err == nil && value > 0 {
		return value, nil
	} else {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
}

func (environment Environment) Duration(key, fallback string) (time.Duration, error) {
	value, err := time.ParseDuration(environment.Get(key, fallback))
	if err == nil && value > 0 {
		return value, nil
	} else {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
}

func Prefix(service string) string {
	return strings.ToUpper(strings.ReplaceAll(service, "-", "_"))
}
