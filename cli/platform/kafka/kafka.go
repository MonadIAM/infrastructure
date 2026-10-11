package kafka

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	broker "github.com/segmentio/kafka-go"

	"github.com/MonadIAM/infrastructure/cli/config"
)

func topics(environment config.Environment) ([]Topic, error) {
	var result []Topic

	for _, definition := range topicDefinitions() {
		topic, err := resolveTopic(environment, definition)
		if err != nil {
			return nil, err
		}

		result = append(result, topic)
		if definition.DeadLetter {
			result = append(result, Topic{
				Name:       topic.Name + "-dead",
				Partitions: 1,
				Config:     retentionConfig(deadLetterRetention),
			})
		}
	}

	return result, nil
}

func resolveTopic(environment config.Environment, definition topicDefinition) (Topic, error) {
	topic := Topic{
		Name:       definition.Name,
		Partitions: 1,
		Grow:       definition.Grow,
		Config:     definition.Config,
	}

	if definition.NameKey != "" {
		err := environment.Require(definition.NameKey)
		if err != nil {
			return topic, err
		}

		topic.Name = environment[definition.NameKey]
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

func Reconcile(ctx context.Context, settings Settings, output io.Writer) error {
	err := settings.Validate()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, settings.Timeout)
	defer cancel()

	transport := &broker.Transport{}
	defer transport.CloseIdleConnections()

	client := &broker.Client{
		Addr:      broker.TCP(settings.Address),
		Timeout:   15 * time.Second,
		Transport: transport,
	}

	metadata, err := client.Metadata(ctx, &broker.MetadataRequest{})
	if err != nil {
		return fmt.Errorf("read Kafka metadata: %w", err)
	}

	existing := make(map[string]int)

	for _, topic := range metadata.Topics {
		existing[topic.Name] = len(topic.Partitions)
	}

	err = checkPartitions(settings.Topics, existing)
	if err != nil {
		return err
	}

	for _, topic := range settings.Topics {
		err := applyTopic(ctx, client, topic, existing[topic.Name])
		if err != nil {
			return err
		}
	}

	_, err = fmt.Fprintf(output, "Configured %d Kafka topics.\n", len(settings.Topics))
	return err
}

func checkPartitions(topics []Topic, existing map[string]int) error {
	controlled := make(map[string]bool)

	for _, definition := range topicDefinitions() {
		controlled[definition.Name] = definition.PartitionsKey != ""
	}

	for _, topic := range topics {
		count := existing[topic.Name]
		if count == 0 || !controlled[topic.Name] || count == topic.Partitions {
			continue
		}

		if topic.Grow && count < topic.Partitions {
			continue
		}

		return fmt.Errorf("%s has %d partitions; expected %d; existing partitions are not changed", topic.Name, count, topic.Partitions)
	}

	return nil
}

func applyTopic(ctx context.Context, client *broker.Client, topic Topic, existing int) error {
	if existing == 0 {
		entries := make([]broker.ConfigEntry, 0, len(topic.Config))

		for key, value := range topic.Config {
			entries = append(entries, broker.ConfigEntry{
				ConfigName:  key,
				ConfigValue: value,
			})
		}

		response, err := client.CreateTopics(ctx, &broker.CreateTopicsRequest{Topics: []broker.TopicConfig{{
			Topic:             topic.Name,
			NumPartitions:     topic.Partitions,
			ReplicationFactor: 1,
			ConfigEntries:     entries,
		}}})
		if err != nil {
			return err
		}

		err = response.Errors[topic.Name]
		if err != nil && !errors.Is(err, broker.TopicAlreadyExists) {
			return err
		}
	}

	if topic.Grow && existing > 0 && existing < topic.Partitions {
		response, err := client.CreatePartitions(ctx, &broker.CreatePartitionsRequest{Topics: []broker.TopicPartitionsConfig{{
			Name:  topic.Name,
			Count: int32(topic.Partitions),
		}}})
		if err == nil {
			return response.Errors[topic.Name]
		} else {
			return err
		}
	}

	return nil
}
