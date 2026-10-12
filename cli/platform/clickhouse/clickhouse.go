package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/MonadIAM/infrastructure/cli/config"
	infrahttp "github.com/MonadIAM/infrastructure/cli/platform/http"
)

type Client struct {
	settings Settings
	http     *resty.Client
}

func New(settings Settings, writer bool) *Client {
	client := infrahttp.New(settings.Address)
	client.SetBasicAuth(settings.User, settings.Password)
	if writer {
		client.SetBasicAuth(settings.WriterUser, settings.WriterPassword)
	}

	client.SetTimeout(settings.Timeout)
	return &Client{settings: settings, http: client}
}

func (client *Client) Query(ctx context.Context, query string, parameters map[string]string) (string, error) {
	fields := map[string]string{"query": query}
	escape := strings.NewReplacer("\\", "\\\\", "\t", "\\t", "\n", "\\n", "\r", "\\r", "\x00", "\\0")
	for name, value := range parameters {
		fields["param_"+name] = escape.Replace(value)
	}

	response, err := client.http.R().
		SetContext(ctx).
		SetMultipartFormData(fields).
		Post("/?wait_end_of_query=1&log_queries=0")
	if err != nil {
		return "", fmt.Errorf("clickhouse.Client.Query: %w", infrahttp.RequestError{Cause: err})
	}

	if response.IsSuccess() {
		return strings.TrimSpace(response.String()), nil
	} else {
		return "", fmt.Errorf("clickhouse.Client.Query: HTTP status %d", response.StatusCode())
	}
}

func (client *Client) Statements() ([]string, error) {
	err := client.settings.Validate()
	if err != nil {
		return nil, err
	}

	files, err := filepath.Glob(filepath.Join(client.settings.SchemaDir, "*.sql"))
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, errors.New("ClickHouse schema contains no SQL files")
	}

	replace := strings.NewReplacer(
		"{{writer_user}}", "`"+client.settings.WriterUser+"`",
		"{{reader_user}}", "`"+client.settings.ReaderUser+"`",
		"'{{writer_password}}'", Quote(client.settings.WriterPassword),
		"'{{reader_password}}'", Quote(client.settings.ReaderPassword),
	)
	var result []string

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}

		statements, err := splitStatements(string(data))
		if err != nil {
			return nil, err
		}

		for _, statement := range statements {
			result = append(result, replace.Replace(statement))
		}
	}

	return result, nil
}

func (client *Client) Reconcile(ctx context.Context, output io.Writer) error {
	statements, err := client.Statements()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, client.settings.Timeout)
	defer cancel()

	for index, statement := range statements {
		_, err := client.Query(ctx, statement, nil)
		if err != nil {
			return fmt.Errorf("clickhouse.Client.Reconcile: ClickHouse schema statement %d: %w", index+1, err)
		}
	}

	_, err = fmt.Fprintf(output, "Applied %d ClickHouse schema statements.\n", len(statements))
	return err
}

func Load(environment config.Environment) (Settings, error) {
	settings := Settings{
		Address:   "http://clickhouse:8123",
		SchemaDir: "audit-archive/clickhouse/schema",
		Timeout:   2 * time.Minute,
	}

	err := environment.Decode(&settings)
	if err != nil {
		return settings, err
	}

	return settings, settings.Validate()
}

func Quote(value string) string {
	escaped := strings.NewReplacer("\\", "\\\\", "'", "\\'", "\n", "\\n", "\r", "\\r", "\x00", "\\0").Replace(value)
	return "'" + escaped + "'"
}

func splitStatements(source string) ([]string, error) {
	var result []string
	var quote rune
	escaped := false
	start := 0

	for index, character := range source {
		if escaped {
			escaped = false
			continue
		}

		if character == '\\' && quote != 0 {
			escaped = true
			continue
		}

		if quote != 0 {
			if character == quote {
				quote = 0
			}

			continue
		}

		switch character {
		case '\'', '"', '`':
			quote = character
		case ';':
			result = appendStatement(result, source[start:index])
			start = index + 1
		}
	}

	if quote != 0 {
		return nil, errors.New("unterminated quote in ClickHouse schema")
	}

	result = appendStatement(result, source[start:])
	return result, nil
}

func appendStatement(statements []string, source string) []string {
	statement := strings.TrimSpace(source)
	if statement != "" {
		return append(statements, statement)
	} else {
		return statements
	}
}
