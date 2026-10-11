package clickhouse

import (
	"errors"
	"regexp"
	"time"

	"github.com/MonadIAM/infrastructure/cli/config"
	infrahttp "github.com/MonadIAM/infrastructure/cli/platform/http"
)

type Settings struct {
	Address        string
	User           string
	Password       string
	WriterUser     string
	WriterPassword string
	ReaderUser     string
	ReaderPassword string
	SchemaDir      string
	Timeout        time.Duration
}

func Load(environment config.Environment) (Settings, error) {
	settings := Settings{
		Address:        environment.Get("CLICKHOUSE_URL", "http://clickhouse:8123"),
		User:           environment["CLICKHOUSE_USER"],
		Password:       environment["CLICKHOUSE_PASSWORD"],
		WriterUser:     environment["CLICKHOUSE_WRITER_USER"],
		WriterPassword: environment["CLICKHOUSE_WRITER_PASSWORD"],
		ReaderUser:     environment["CLICKHOUSE_READER_USER"],
		ReaderPassword: environment["CLICKHOUSE_READER_PASSWORD"],
		SchemaDir:      environment.Get("CLICKHOUSE_SCHEMA_DIR", "audit-archive/clickhouse/schema"),
	}
	var err error
	settings.Timeout, err = environment.Duration("INFRA_TIMEOUT", "2m")
	if err == nil {
		return settings, settings.Validate()
	} else {
		return settings, err
	}
}

func (settings Settings) Validate() error {
	err := infrahttp.ValidateURL(settings.Address)
	if err != nil {
		return err
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
