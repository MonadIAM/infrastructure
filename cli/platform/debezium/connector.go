package debezium

import (
	"errors"
	"fmt"
	"strings"
)

type Connector struct {
	Name   string
	Config map[string]string
}

func (connector Connector) Validate() error {
	if connector.Name == "" || strings.ContainsAny(connector.Name, "/?#% \t\r\n") {
		return errors.New("connector name must be a nonempty path segment")
	}

	required := []string{
		"connector.class",
		"database.hostname",
		"database.port",
		"database.user",
		"database.password",
		"database.dbname",
		"slot.name",
		"publication.name",
	}

	for _, key := range required {
		if connector.Config[key] == "" {
			return fmt.Errorf("debezium.Connector.Validate: connector %s requires %s", connector.Name, key)
		}
	}

	return nil
}
