package kafka

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

type Settings struct {
	Address string        `koanf:"KAFKA_BOOTSTRAP_SERVER"`
	Timeout time.Duration `koanf:"INFRA_TIMEOUT"`
	Topics  []Topic       `koanf:"-"`
}

type Topic struct {
	Name       string
	Partitions int
	Grow       bool
	Config     map[string]string
}

var topicNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,249}$`)

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
			return fmt.Errorf("kafka.Settings.Validate: %s needs positive partitions", topic.Name)
		}

		if seen[topic.Name] {
			return fmt.Errorf("kafka.Settings.Validate: duplicate Kafka topic %s", topic.Name)
		}

		seen[topic.Name] = true
	}

	return nil
}
