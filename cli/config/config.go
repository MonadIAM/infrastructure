package config

import (
	"errors"
	"fmt"
	"io"
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

func Initialize(path string) error {
	if path == "" {
		return errors.New("env file path must not be empty")
	}

	_, err := os.Stat(path)
	if err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	source, err := os.Open(".env.example")
	if err != nil {
		return err
	}
	defer func() {
		_ = source.Close()
	}()

	destination, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() {
		_ = destination.Close()
	}()

	_, err = io.Copy(destination, source)
	if err != nil {
		return err
	}

	return destination.Sync()
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

func Validate(environment Environment) error {
	err := environment.Require(required...)
	if err != nil {
		return err
	}

	keys := []string{
		"HELM_RETRY_ATTEMPTS",
		"HELM_RETRY_DELAY_SECONDS",
		"HELM_RETRY_MAX_DELAY_SECONDS",
		"VAULT_KEY_SHARES",
		"VAULT_KEY_THRESHOLD",
		"PROJECTION_JOURNAL_PARTITIONS",
	}

	for _, key := range keys {
		_, err := environment.PositiveInt(key)
		if err != nil {
			return err
		}
	}

	shares, _ := environment.PositiveInt("VAULT_KEY_SHARES")
	threshold, _ := environment.PositiveInt("VAULT_KEY_THRESHOLD")
	if threshold > shares {
		return fmt.Errorf("VAULT_KEY_THRESHOLD must not exceed VAULT_KEY_SHARES")
	}

	_, err = environment.Duration("INFRA_TIMEOUT", "60m")

	return err
}
