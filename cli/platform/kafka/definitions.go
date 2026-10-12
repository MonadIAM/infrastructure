package kafka

import (
	"maps"
	"strconv"
	"time"

	"github.com/MonadIAM/infrastructure/cli/config"
)

type topicDefinition struct {
	Retention     time.Duration
	Name          string
	NameKey       string
	PartitionsKey string
	Grow          bool
	DeadLetter    bool
	Config        map[string]string
}

const (
	defaultTopicRetention     = 72 * time.Hour
	deadLetterRetention       = 30 * 24 * time.Hour
	retryTopicRetention       = 7 * 24 * time.Hour
	reauthenticationRetention = 30 * time.Minute
	blacklistRetention        = 15 * time.Minute
)

var topicDefinitions = []topicDefinition{
	{
		Name:          "realm",
		PartitionsKey: "REALM_PARTITIONS",
		DeadLetter:    true,
		Retention:     defaultTopicRetention,
	},
	{
		Name:          "position",
		PartitionsKey: "POSITION_PARTITIONS",
		DeadLetter:    true,
		Retention:     defaultTopicRetention,
	},
	{
		Name:       "account",
		DeadLetter: true,
		Retention:  defaultTopicRetention,
	},
	{
		Name:       "identifier",
		DeadLetter: true,
		Retention:  defaultTopicRetention,
	},
	{
		Name:       "service-client",
		DeadLetter: true,
		Retention:  defaultTopicRetention,
	},
	{
		Name:       "interface-client",
		DeadLetter: true,
		Retention:  defaultTopicRetention,
	},
	{
		Name:       "notification",
		DeadLetter: true,
		Retention:  defaultTopicRetention,
	},
	{
		Name:       "message-dispatch",
		DeadLetter: true,
		Retention:  defaultTopicRetention,
	},
	{
		Name:       "access-cache",
		DeadLetter: true,
		Retention:  defaultTopicRetention,
	},
	{
		Name:          "projection-journal",
		PartitionsKey: "PROJECTION_JOURNAL_PARTITIONS",
		Grow:          true,
		DeadLetter:    true,
		Retention:     defaultTopicRetention,
	},
	{
		Name:      "message-dispatch-retry-notification-service",
		Retention: retryTopicRetention,
	},
	{
		Name:      "reauthentication",
		Retention: reauthenticationRetention,
	},
	{
		Name:      "reauthentication-dead",
		Retention: deadLetterRetention,
	},
	{
		Name:      "blacklist",
		Retention: blacklistRetention,
		Config: map[string]string{
			"cleanup.policy":            "compact,delete",
			"min.cleanable.dirty.ratio": "0.01",
			"segment.ms":                strconv.FormatInt(time.Minute.Milliseconds(), 10),
		},
	},
	{
		Name:      "blacklist-dead",
		Retention: retryTopicRetention,
	},
	{
		Name:      "audit-log-archive",
		Retention: defaultTopicRetention,
	},
	{
		Name:      "change-log-archive",
		Retention: defaultTopicRetention,
	},
	{
		NameKey: "KAFKA_CONNECT_CONFIG_STORAGE_TOPIC",
		Config:  map[string]string{"cleanup.policy": "compact"},
	},
	{
		NameKey: "KAFKA_CONNECT_OFFSET_STORAGE_TOPIC",
		Config:  map[string]string{"cleanup.policy": "compact"},
	},
	{
		NameKey: "KAFKA_CONNECT_STATUS_STORAGE_TOPIC",
		Config:  map[string]string{"cleanup.policy": "compact"},
	},
	{
		NameKey: "SCHEMA_REGISTRY_KAFKASQL_JOURNAL_TOPIC",
		Config: map[string]string{
			"cleanup.policy":  "delete",
			"retention.ms":    "-1",
			"retention.bytes": "-1",
		},
	},
	{
		NameKey: "SCHEMA_REGISTRY_KAFKASQL_SNAPSHOTS_TOPIC",
		Config: map[string]string{
			"cleanup.policy":  "delete",
			"retention.ms":    "-1",
			"retention.bytes": "-1",
		},
	},
	{
		NameKey: "SCHEMA_REGISTRY_EVENTS_TOPIC",
		Config: map[string]string{
			"cleanup.policy":  "delete",
			"retention.ms":    "-1",
			"retention.bytes": "-1",
		},
	},
}

func (definition topicDefinition) resolve(environment config.Environment) (Topic, error) {
	topic := Topic{
		Name:       definition.Name,
		Partitions: 1,
		Grow:       definition.Grow,
		Config:     make(map[string]string, len(definition.Config)+1),
	}

	maps.Copy(topic.Config, definition.Config)
	if definition.Retention > 0 {
		topic.Config["retention.ms"] = strconv.FormatInt(definition.Retention.Milliseconds(), 10)
	}

	if definition.NameKey != "" {
		err := environment.Require(definition.NameKey)
		if err != nil {
			return topic, err
		}

		topic.Name = environment.String(definition.NameKey)
	}

	if definition.PartitionsKey != "" {
		count, err := environment.PositiveInt(definition.PartitionsKey)
		if err != nil {
			return topic, err
		}

		topic.Partitions = count
	}

	return topic, nil
}
