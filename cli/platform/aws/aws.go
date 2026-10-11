package aws

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func Import(csvPath, envPath string, output io.Writer) error {
	credentials, err := readCSV(csvPath)
	if err != nil {
		return err
	}

	original, err := os.ReadFile(envPath)
	if err != nil {
		return errors.New("cannot read target environment file")
	}

	lines := strings.Split(string(original), "\n")
	found := make(map[string]bool)

	for index, line := range lines {
		key, _, exists := strings.Cut(line, "=")
		value, ok := credentials[key]
		if ok && exists {
			lines[index] = key + "='" + value + "'"
			found[key] = true
		}
	}

	credentialKeys := []string{"NOTIFICATION_AWS_ACCESS_KEY_ID", "NOTIFICATION_AWS_SECRET_ACCESS_KEY"}

	for _, key := range credentialKeys {
		if !found[key] {
			lines = append(lines, key+"='"+credentials[key]+"'")
		}
	}

	err = replaceEnvironment(envPath, []byte(strings.Join(lines, "\n")))
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(output, "Notification AWS credentials imported.")
	return err
}

func readCSV(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New("cannot read AWS credentials CSV")
	}

	defer func() { _ = file.Close() }()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil || len(records) != 2 {
		return nil, errors.New("AWS CSV must contain a header and exactly one credential row")
	}

	values := make(map[string]string)

	for index, header := range records[0] {
		switch strings.ToLower(strings.TrimSpace(strings.TrimPrefix(header, "\ufeff"))) {
		case "access key id":
			values["NOTIFICATION_AWS_ACCESS_KEY_ID"] = records[1][index]
		case "secret access key":
			values["NOTIFICATION_AWS_SECRET_ACCESS_KEY"] = records[1][index]
		}
	}

	allowed := regexp.MustCompile(`^[A-Za-z0-9/+=]+$`)

	credentialKeys := []string{"NOTIFICATION_AWS_ACCESS_KEY_ID", "NOTIFICATION_AWS_SECRET_ACCESS_KEY"}

	for _, key := range credentialKeys {
		if !allowed.MatchString(values[key]) {
			return nil, errors.New("AWS CSV must contain valid access key ID and secret access key columns")
		}
	}

	return values, nil
}

func replaceEnvironment(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".env-*")
	if err != nil {
		return err
	}

	defer func() { _ = os.Remove(file.Name()) }()

	_, err = file.Write(data)
	if err != nil {
		_ = file.Close()
		return err
	}

	err = file.Sync()
	if err != nil {
		_ = file.Close()
		return err
	}

	err = file.Close()
	if err == nil {
		return os.Rename(file.Name(), path)
	} else {
		return err
	}
}
