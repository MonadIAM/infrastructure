package archive

import (
	"errors"
	"net/url"
	"time"

	"github.com/MonadIAM/infrastructure/cli/platform/clickhouse"
	"github.com/MonadIAM/infrastructure/cli/platform/seaweedfs"
)

type Settings struct {
	ClickHouse clickhouse.Settings
	Storage    seaweedfs.Settings
	QueryDir   string
	AfterDays  int
	Timeout    time.Duration
}

type table struct {
	Name   string
	Bucket string
}

type partitionState struct {
	exists   bool
	archived uint64
	missing  uint64
}

func (settings Settings) Validate() error {
	err := settings.ClickHouse.Validate()
	if err != nil {
		return err
	}

	endpoint, err := url.Parse(settings.Storage.Endpoint)
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") {
		return errors.New("invalid SeaweedFS endpoint")
	}

	if settings.Storage.ArchiverKey == "" || settings.Storage.ArchiverSecret == "" ||
		len(settings.Storage.Buckets) != 2 || settings.Storage.Region == "" {
		return errors.New("archive storage credentials, region and buckets are required")
	}

	if settings.AfterDays < 1 || settings.Timeout <= 0 || settings.QueryDir == "" {
		return errors.New("archive age, timeout and query directory are required")
	} else {
		return nil
	}
}
