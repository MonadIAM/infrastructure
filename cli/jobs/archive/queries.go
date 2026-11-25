package archive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func loadQueries(directory string, tables []table) (map[string]string, error) {
	names := []string{
		"partitions",
		"latest",
		"source-count",
		"archive-count",
		"missing-count",
		"record",
	}
	for _, table := range tables {
		names = append(names, "export-"+table.Name, "merge-"+table.Name)
	}

	queries := make(map[string]string, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(directory, name+".sql"))
		if err != nil {
			return nil, fmt.Errorf("read archive query %s: %w", name, err)
		}

		query := strings.TrimSpace(string(data))
		if query == "" {
			return nil, errors.New("empty archive query: " + name)
		}

		queries[name] = query
	}

	return queries, nil
}
