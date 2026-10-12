package archive

import (
	"errors"
	"time"

	"github.com/MonadIAM/infrastructure/cli/platform/clickhouse"
	"github.com/MonadIAM/infrastructure/cli/platform/seaweedfs"
)

type Settings struct {
	ClickHouse clickhouse.Settings `koanf:"-"`
	Storage    seaweedfs.Settings  `koanf:"-"`
	AfterDays  int                 `koanf:"ARCHIVE_AFTER_DAYS"`
	Interval   time.Duration       `koanf:"-"`
	Timeout    time.Duration       `koanf:"ARCHIVE_TIMEOUT"`
}

type table struct {
	Name       string
	Bucket     string
	Projection string
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
