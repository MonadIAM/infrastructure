package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	infrahttp "github.com/MonadIAM/infrastructure/cli/platform/http"
)

func Client(settings Settings, writer bool) infrahttp.Client {
	client := infrahttp.New(settings.Address)
	client.User, client.Password = settings.User, settings.Password
	if writer {
		client.User, client.Password = settings.WriterUser, settings.WriterPassword
	}

	client.HTTP.Timeout = settings.Timeout
	return client
}

func Query(ctx context.Context, client infrahttp.Client, query string) (string, error) {
	body, err := client.Do(ctx, http.MethodPost, "/?wait_end_of_query=1&log_queries=0", []byte(query))
	if err == nil {
		return strings.TrimSpace(string(body)), nil
	} else {
		return "", err
	}
}

func Quote(value string) string {
	escaped := strings.NewReplacer("\\", "\\\\", "'", "\\'", "\n", "\\n", "\r", "\\r", "\x00", "\\0").Replace(value)
	return "'" + escaped + "'"
}

func Statements(settings Settings) ([]string, error) {
	err := settings.Validate()
	if err != nil {
		return nil, err
	}

	files, err := filepath.Glob(filepath.Join(settings.SchemaDir, "*.sql"))
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, errors.New("ClickHouse schema contains no SQL files")
	}

	replace := strings.NewReplacer(
		"{{writer_user}}", "`"+settings.WriterUser+"`",
		"{{reader_user}}", "`"+settings.ReaderUser+"`",
		"'{{writer_password}}'", Quote(settings.WriterPassword),
		"'{{reader_password}}'", Quote(settings.ReaderPassword),
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

func Reconcile(ctx context.Context, settings Settings, output io.Writer) error {
	statements, err := Statements(settings)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, settings.Timeout)
	defer cancel()

	client := Client(settings, false)

	for index, statement := range statements {
		_, err := Query(ctx, client, statement)
		if err != nil {
			return fmt.Errorf("ClickHouse schema statement %d: %w", index+1, err)
		}
	}

	_, err = fmt.Fprintf(output, "Applied %d ClickHouse schema statements.\n", len(statements))
	return err
}

func appendStatement(statements []string, source string) []string {
	statement := strings.TrimSpace(source)
	if statement != "" {
		return append(statements, statement)
	} else {
		return statements
	}
}
