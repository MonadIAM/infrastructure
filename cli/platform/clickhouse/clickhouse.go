package clickhouse

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MonadIAM/infrastructure/cli/config"
)

type Settings struct {
	Address   string
	User      string
	Password  string
	SchemaDir string
	Timeout   time.Duration
}

type Client struct {
	settings Settings
	http     *http.Client
}

func Load(environment config.Environment) (Settings, error) {
	timeout, err := environment.Duration("INFRA_TIMEOUT", "2m")
	if err != nil {
		return Settings{}, err
	}

	settings := Settings{
		Address:   environment.Get("CLICKHOUSE_URL", "http://clickhouse-main.monadiam-data.svc:8123"),
		User:      environment["CLICKHOUSE_USER"],
		Password:  environment["CLICKHOUSE_PASSWORD"],
		SchemaDir: environment.Get("CLICKHOUSE_SCHEMA_DIR", "/schema"),
		Timeout:   timeout,
	}

	return settings, settings.Validate()
}

func (settings Settings) Validate() error {
	address, err := url.Parse(settings.Address)
	if err != nil || address.Host == "" || (address.Scheme != "http" && address.Scheme != "https") {
		return errors.New("invalid ClickHouse URL")
	}

	if settings.User == "" || settings.Password == "" || settings.Timeout <= 0 {
		return errors.New("ClickHouse credentials and positive timeout are required")
	}

	return nil
}

func New(settings Settings) *Client {
	return &Client{settings: settings, http: &http.Client{Timeout: settings.Timeout}}
}

func (client *Client) Query(ctx context.Context, query string, parameters map[string]string) (string, error) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	fields := map[string]string{"query": query}
	for key, value := range parameters {
		fields["param_"+key] = value
	}

	for key, value := range fields {
		err := form.WriteField(key, value)
		if err != nil {
			return "", err
		}
	}

	err := form.Close()
	if err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(client.settings.Address, "/")+"/?wait_end_of_query=1&log_queries=0", &body)
	if err != nil {
		return "", err
	}

	request.SetBasicAuth(client.settings.User, client.settings.Password)
	request.Header.Set("Content-Type", form.FormDataContentType())
	response, err := client.http.Do(request)
	if err != nil {
		return "", err
	}

	result, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil {
		return "", errors.Join(readErr, closeErr)
	}

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ClickHouse HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(result)))
	}

	return strings.TrimSpace(string(result)), nil
}

func (client *Client) Reconcile(ctx context.Context, output io.Writer) error {
	paths, err := filepath.Glob(filepath.Join(client.settings.SchemaDir, "*.sql"))
	if err != nil || len(paths) == 0 {
		return errors.New("ClickHouse schema contains no SQL files")
	}

	count := 0
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		for _, statement := range strings.Split(string(data), ";") {
			statement = strings.TrimSpace(statement)
			if statement == "" {
				continue
			}

			_, err = client.Query(ctx, statement, nil)
			if err != nil {
				return fmt.Errorf("ClickHouse schema %s statement %d: %w", filepath.Base(path), count+1, err)
			}

			count++
		}
	}

	_, err = fmt.Fprintf(output, "Applied %d ClickHouse schema statements.\n", count)
	return err
}
