package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Environment struct {
	*koanf.Koanf
}

func (environment Environment) ParseDebezium() (Debezium, error) {
	settings := Debezium{
		ConnectURL:     "http://kafka-connect:8083",
		TemplatePath:   "messaging/debezium/outbox.connector.json",
		Timeout:        2 * time.Minute,
		RequestTimeout: 10 * time.Second,
		RetryInterval:  2 * time.Second,
	}

	err := environment.Decode(&settings)
	if err != nil {
		return settings, err
	}

	for _, service := range Services() {
		settings.Databases = append(settings.Databases, environment.readDatabase(service))
	}

	return settings, settings.Validate()
}

func (environment Environment) Get(key, fallback string) string {
	if environment.Exists(key) {
		return environment.String(key)
	} else {
		return fallback
	}
}

func (environment Environment) Require(keys ...string) error {
	for _, key := range keys {
		if environment.String(key) == "" {
			return fmt.Errorf("config.Environment.Require: %s is required", key)
		}
	}

	return nil
}

func (environment Environment) PositiveInt(key string) (int, error) {
	value, err := strconv.Atoi(environment.String(key))
	if err == nil && value > 0 {
		return value, nil
	} else {
		return 0, fmt.Errorf("config.Environment.PositiveInt: %s must be a positive integer", key)
	}
}

func (environment Environment) Decode(target any) error {
	err := environment.Unmarshal("", target)
	if err == nil {
		return nil
	} else {
		return fmt.Errorf("config.Environment.Decode: %w", sourceError{cause: err})
	}
}

func (environment Environment) readDatabase(service string) Database {
	prefix := strings.ToUpper(strings.ReplaceAll(service, "-", "_")) + "_POSTGRES_"

	return Database{
		Service:     service + "-service",
		Host:        service + "-postgresql",
		Name:        environment.String(prefix + "DB"),
		User:        environment.String(prefix + "USER"),
		Password:    environment.String(prefix + "PASSWORD"),
		Slot:        environment.String(prefix + "OUTBOX_SLOT"),
		Publication: environment.String(prefix + "OUTBOX_PUBLICATION"),
	}
}

func Load(path string, overrides map[string]string) (Environment, error) {
	settings, err := readFile(path)
	if err != nil {
		return Environment{}, fmt.Errorf("config.Load: %w", err)
	}

	err = settings.Load(env.Provider("", "", nil), nil)
	if err != nil {
		return Environment{}, fmt.Errorf("config.Load: environment: %w", sourceError{cause: err})
	}

	for key, value := range overrides {
		err := settings.Set(key, value)
		if err != nil {
			return Environment{}, fmt.Errorf("config.Load: override %s: %w", key, sourceError{cause: err})
		}
	}

	return Environment{Koanf: settings}, nil
}

func ReadFile(path string) (Environment, error) {
	settings, err := readFile(path)
	if err == nil {
		return Environment{Koanf: settings}, nil
	} else {
		return Environment{}, fmt.Errorf("config.ReadFile: %w", err)
	}
}

func readFile(path string) (*koanf.Koanf, error) {
	settings := koanf.New(".")
	if path == "" {
		return settings, nil
	}

	err := settings.Load(file.Provider(path), dotenv.Parser())
	if err == nil {
		return settings, nil
	} else {
		return nil, fmt.Errorf("config.readFile: environment file %s: %w", path, sourceError{cause: err})
	}
}
