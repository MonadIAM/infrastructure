package kafka

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	broker "github.com/segmentio/kafka-go"

	"github.com/MonadIAM/infrastructure/cli/config"
)

type Service struct {
	settings Settings
	client   *broker.Client
}

func New(settings Settings) *Service {
	return &Service{settings: settings}
}

func (service *Service) Reconcile(ctx context.Context, output io.Writer) error {
	err := service.settings.Validate()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, service.settings.Timeout)
	defer cancel()

	transport := &broker.Transport{}
	defer transport.CloseIdleConnections()

	service.client = &broker.Client{
		Addr:      broker.TCP(service.settings.Address),
		Timeout:   15 * time.Second,
		Transport: transport,
	}

	metadata, err := service.client.Metadata(ctx, &broker.MetadataRequest{})
	if err != nil {
		return fmt.Errorf("kafka.Service.Reconcile: read Kafka metadata: %w", err)
	}

	existing := make(map[string]int)

	for _, topic := range metadata.Topics {
		existing[topic.Name] = len(topic.Partitions)
	}

	err = service.checkPartitions(existing)
	if err != nil {
		return err
	}

	for _, topic := range service.settings.Topics {
		err := service.applyTopic(ctx, topic, existing[topic.Name])
		if err != nil {
			return err
		}
	}

	_, err = fmt.Fprintf(output, "Configured %d Kafka topics.\n", len(service.settings.Topics))
	return err
}

func (service *Service) checkPartitions(existing map[string]int) error {
	controlled := make(map[string]bool)

	for _, definition := range topicDefinitions {
		controlled[definition.Name] = definition.PartitionsKey != ""
	}

	for _, topic := range service.settings.Topics {
		count := existing[topic.Name]
		if count == 0 || !controlled[topic.Name] || count == topic.Partitions {
			continue
		}

		if topic.Grow && count < topic.Partitions {
			continue
		}

		return fmt.Errorf("kafka.Service.checkPartitions: %s has %d partitions; expected %d; existing partitions are not changed", topic.Name, count, topic.Partitions)
	}

	return nil
}

func (service *Service) applyTopic(ctx context.Context, topic Topic, existing int) error {
	if existing == 0 {
		entries := make([]broker.ConfigEntry, 0, len(topic.Config))

		for key, value := range topic.Config {
			entries = append(entries, broker.ConfigEntry{
				ConfigName:  key,
				ConfigValue: value,
			})
		}

		response, err := service.client.CreateTopics(ctx, &broker.CreateTopicsRequest{Topics: []broker.TopicConfig{{
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
		response, err := service.client.CreatePartitions(ctx, &broker.CreatePartitionsRequest{Topics: []broker.TopicPartitionsConfig{{
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

func Load(environment config.Environment) (Settings, error) {
	settings := Settings{
		Address: "kafka:29092",
		Timeout: 2 * time.Minute,
	}

	err := environment.Decode(&settings)
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

func topics(environment config.Environment) ([]Topic, error) {
	var result []Topic

	for _, definition := range topicDefinitions {
		topic, err := definition.resolve(environment)
		if err != nil {
			return nil, err
		}

		result = append(result, topic)
		if definition.DeadLetter {
			result = append(result, Topic{
				Name:       topic.Name + "-dead",
				Partitions: 1,
				Config: map[string]string{
					"retention.ms": strconv.FormatInt(deadLetterRetention.Milliseconds(), 10),
				},
			})
		}
	}

	return result, nil
}
