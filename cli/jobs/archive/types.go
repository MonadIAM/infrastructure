package archive

import (
	"errors"
	"time"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/clickhouse"
	"github.com/MonadIAM/infrastructure/cli/platform/seaweedfs"
)

type Settings struct {
	ClickHouse clickhouse.Settings
	Storage    seaweedfs.Settings
	AfterDays  int
	Interval   time.Duration
	Timeout    time.Duration
}

type table struct {
	Name       string
	Bucket     string
	Projection string
}

func Load(environment config.Environment) (Settings, error) {
	var settings Settings
	var err error
	settings.ClickHouse, err = clickhouse.Load(environment)
	if err != nil {
		return settings, err
	}

	settings.Storage, err = seaweedfs.Load(environment)
	if err != nil {
		return settings, err
	}

	settings.AfterDays, err = environment.PositiveInt("ARCHIVE_AFTER_DAYS")
	if err != nil {
		return settings, err
	}

	seconds, err := environment.PositiveInt("ARCHIVE_INTERVAL_SECONDS")
	if err != nil {
		return settings, err
	}

	if seconds > int((24*time.Hour*365)/time.Second) {
		return settings, errors.New("archive interval must not exceed one year")
	}

	settings.Interval = time.Duration(seconds) * time.Second
	settings.Timeout, err = environment.Duration("ARCHIVE_TIMEOUT", "30m")
	if err == nil {
		return settings, settings.Validate()
	} else {
		return settings, err
	}
}

func (settings Settings) Validate() error {
	err := settings.ClickHouse.Validate()
	if err != nil {
		return err
	}

	err = settings.Storage.Validate()
	if err != nil {
		return err
	}

	if settings.AfterDays < 1 || settings.Interval <= 0 || settings.Timeout <= 0 {
		return errors.New("archive age, interval and timeout must be positive")
	} else {
		return nil
	}
}
