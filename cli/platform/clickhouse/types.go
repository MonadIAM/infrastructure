package clickhouse

import (
	"errors"
	"regexp"
	"time"
)

type Settings struct {
	Address        string        `koanf:"CLICKHOUSE_URL"`
	User           string        `koanf:"CLICKHOUSE_USER"`
	Password       string        `koanf:"CLICKHOUSE_PASSWORD"`
	WriterUser     string        `koanf:"CLICKHOUSE_WRITER_USER"`
	WriterPassword string        `koanf:"CLICKHOUSE_WRITER_PASSWORD"`
	ReaderUser     string        `koanf:"CLICKHOUSE_READER_USER"`
	ReaderPassword string        `koanf:"CLICKHOUSE_READER_PASSWORD"`
	SchemaDir      string        `koanf:"CLICKHOUSE_SCHEMA_DIR"`
	Timeout        time.Duration `koanf:"INFRA_TIMEOUT"`
}

func (settings Settings) Validate() error {
	if settings.Address == "" {
		return errors.New("ClickHouse address is required")
	}

	valid := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	if !valid.MatchString(settings.WriterUser) || !valid.MatchString(settings.ReaderUser) {
		return errors.New("invalid ClickHouse writer or reader username")
	}

	if settings.User == "" || settings.Password == "" || settings.WriterPassword == "" || settings.ReaderPassword == "" {
		return errors.New("ClickHouse credentials are required")
	}

	if settings.SchemaDir == "" || settings.Timeout <= 0 {
		return errors.New("ClickHouse schema directory and positive timeout are required")
	} else {
		return nil
	}
}
