package kafka

import (
	"strconv"
	"time"
)

const (
	defaultTopicRetention     = 72 * time.Hour
	deadLetterRetention       = 30 * 24 * time.Hour
	retryTopicRetention       = 7 * 24 * time.Hour
	reauthenticationRetention = 30 * time.Minute
	blacklistRetention        = 15 * time.Minute
)

func topicDefinitions() []topicDefinition {
	return []topicDefinition{
		{
			Name:          "realm",
			PartitionsKey: "REALM_PARTITIONS",
			DeadLetter:    true,
			Config:        retentionConfig(defaultTopicRetention),
		},
		{
			Name:          "position",
			PartitionsKey: "POSITION_PARTITIONS",
			DeadLetter:    true,
			Config:        retentionConfig(defaultTopicRetention),
		},
		{
			Name:       "account",
			DeadLetter: true,
			Config:     retentionConfig(defaultTopicRetention),
		},
		{
			Name:       "identifier",
			DeadLetter: true,
			Config:     retentionConfig(defaultTopicRetention),
		},
		{
			Name:       "service-client",
			DeadLetter: true,
			Config:     retentionConfig(defaultTopicRetention),
		},
		{
			Name:       "interface-client",
			DeadLetter: true,
			Config:     retentionConfig(defaultTopicRetention),
		},
		{
			Name:       "notification",
			DeadLetter: true,
			Config:     retentionConfig(defaultTopicRetention),
		},
		{
			Name:       "message-dispatch",
			DeadLetter: true,
			Config:     retentionConfig(defaultTopicRetention),
		},
		{
			Name:       "access-cache",
			DeadLetter: true,
			Config:     retentionConfig(defaultTopicRetention),
		},
		{
			Name:          "projection-journal",
			PartitionsKey: "PROJECTION_JOURNAL_PARTITIONS",
			Grow:          true,
			DeadLetter:    true,
			Config:        retentionConfig(defaultTopicRetention),
		},
		{
			Name:   "message-dispatch-retry-notification-service",
			Config: retentionConfig(retryTopicRetention),
		},
		{
			Name:   "reauthentication",
			Config: retentionConfig(reauthenticationRetention),
		},
		{
			Name:   "reauthentication-dead",
			Config: retentionConfig(deadLetterRetention),
		},
		{
			Name: "blacklist",
			Config: map[string]string{
				"cleanup.policy":            "compact,delete",
				"retention.ms":              milliseconds(blacklistRetention),
				"min.cleanable.dirty.ratio": "0.01",
				"segment.ms":                milliseconds(time.Minute),
			},
		},
		{
			Name:   "blacklist-dead",
			Config: retentionConfig(retryTopicRetention),
		},
		{
			Name:   "audit-log-archive",
			Config: retentionConfig(defaultTopicRetention),
		},
		{
			Name:   "change-log-archive",
			Config: retentionConfig(defaultTopicRetention),
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
}

func retentionConfig(retention time.Duration) map[string]string {
	return map[string]string{"retention.ms": milliseconds(retention)}
}

func milliseconds(duration time.Duration) string {
	return strconv.FormatInt(duration.Milliseconds(), 10)
}
