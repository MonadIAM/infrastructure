package debezium

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Connector struct {
	Name   string
	Config map[string]string
}

type connectClient struct {
	http          *http.Client
	endpoint      string
	retryInterval time.Duration
}

type requestError struct {
	status    int
	temporary bool
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
			return fmt.Errorf("connector %s requires %s", connector.Name, key)
		}
	}

	return nil
}

func (failure requestError) Error() string {
	if failure.status == 0 {
		return "Kafka Connect request failed"
	} else {
		return fmt.Sprintf("Kafka Connect returned HTTP %d", failure.status)
	}
}
