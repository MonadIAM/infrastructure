package kafka

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/MonadIAM/infrastructure/cli/config"
)

var topicNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,249}$`)

type Settings struct {
	Address string
	Timeout time.Duration
	Topics  []Topic
}

type Topic struct {
	Name       string
	Partitions int
	Grow       bool
	Config     map[string]string
}

type topicDefinition struct {
	Name          string
	NameKey       string
	PartitionsKey string
	Grow          bool
	DeadLetter    bool
	Config        map[string]string
}

func (settings Settings) Validate() error {
	if settings.Address == "" || settings.Timeout <= 0 {
		return errors.New("kafka address and positive timeout are required")
	}

	seen := make(map[string]bool)

	for _, topic := range settings.Topics {
		if !topicNamePattern.MatchString(topic.Name) || topic.Name == "." || topic.Name == ".." {
			return errors.New("invalid Kafka topic name")
		}

		if topic.Partitions < 1 {
			return fmt.Errorf("%s needs positive partitions", topic.Name)
		}

		if seen[topic.Name] {
			return fmt.Errorf("duplicate Kafka topic %s", topic.Name)
		}

		seen[topic.Name] = true
	}

	return nil
}

func Load(environment config.Environment) (Settings, error) {
	settings := Settings{Address: environment.Get("KAFKA_BOOTSTRAP_SERVER", "kafka:29092")}
	var err error
	settings.Timeout, err = environment.Duration("INFRA_TIMEOUT", "2m")
	if err != nil {
		return settings, err
	}

	settings.Topics, err = topics(environment)
	if err == nil {
		return settings, settings.Validate()
	} else {
		return settings, err
	}
}
