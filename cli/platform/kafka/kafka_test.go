package kafka

import (
	"testing"
	"time"

	"github.com/joho/godotenv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

func TestKafka(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kafka")
}

var _ = Describe("[Infrastructure] - Kafka", func() {
	Describe("[Function] - Load", func() {
		It("[case] - loads partition overrides, dead letters and retention rules", func() {
			// Arrange
			environment, err := godotenv.Read("../../../.env.example")
			Expect(err).NotTo(HaveOccurred())
			environment["REALM_PARTITIONS"] = "7"
			environment["POSITION_PARTITIONS"] = "9"
			environment["PROJECTION_JOURNAL_PARTITIONS"] = "11"

			// Act
			settings, err := Load(environment)

			byName := make(map[string]Topic)

			for _, topic := range settings.Topics {
				byName[topic.Name] = topic
			}

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(byName["realm"].Partitions).To(Equal(7))
			Expect(byName["position"].Partitions).To(Equal(9))
			Expect(byName["projection-journal"].Partitions).To(Equal(11))
			Expect(byName["projection-journal"].Grow).To(BeTrue())
			Expect(byName["realm-dead"].Config).To(HaveKeyWithValue("retention.ms", "2592000000"))
			Expect(byName["position-dead"].Partitions).To(Equal(1))
			Expect(byName["blacklist"].Config["cleanup.policy"]).To(Equal("compact,delete"))
			Expect(byName["kafkasql-journal"].Config["retention.ms"]).To(Equal("-1"))
			Expect(byName).NotTo(HaveKey("realm-retry-identity-service"))
		})

		It("[case] - loads fresh topic definitions after a previous configuration was changed", func() {
			// Arrange
			environment, err := godotenv.Read("../../../.env.example")
			Expect(err).NotTo(HaveOccurred())
			previous, err := Load(environment)
			Expect(err).NotTo(HaveOccurred())
			for index := range previous.Topics {
				previous.Topics[index].Config["retention.ms"] = "1"
				previous.Topics[index].Name = "changed"
			}

			// Act
			current, err := Load(environment)

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(current.Topics).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":   Equal("realm"),
				"Config": HaveKeyWithValue("retention.ms", "259200000"),
			})))
		})

	})

	Describe("[Function] - checkPartitions", func() {
		DescribeTable("[Behavior] - accepts only safe partition counts", func(name string, existing, requested int, grow bool, accepted bool) {
			// Arrange
			topics := []Topic{{
				Name:       name,
				Partitions: requested,
				Grow:       grow,
			}}
			current := map[string]int{name: existing}

			// Act
			err := checkPartitions(topics, current)

			// Assert
			if accepted {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
		},
			Entry("[case] - realm cannot grow", "realm", 2, 4, false, false),
			Entry("[case] - position cannot shrink", "position", 4, 2, false, false),
			Entry("[case] - projection can grow", "projection-journal", 2, 4, true, true),
			Entry("[case] - projection cannot shrink", "projection-journal", 4, 2, true, false),
			Entry("[case] - missing topic can be created", "realm", 0, 4, false, true),
			Entry("[case] - matching partition count is accepted", "realm", 4, 4, false, true),
			Entry("[case] - fixed default does not constrain an existing topic", "account", 4, 1, false, true),
		)
	})

	Describe("[Method] - Settings.Validate", func() {
		DescribeTable("[Behavior] - rejects invalid topic declarations", func(topics []Topic) {
			// Arrange
			settings := Settings{
				Address: "kafka:29092",
				Timeout: time.Minute,
				Topics:  topics,
			}

			// Act
			err := settings.Validate()

			// Assert
			Expect(err).NotTo(Succeed())
		},
			Entry("[case] - path separator", []Topic{{
				Name:       "bad/name",
				Partitions: 1,
			}}),
			Entry("[case] - zero partitions", []Topic{{
				Name:       "realm",
				Partitions: 0,
			}}),
			Entry("[case] - duplicate", []Topic{{
				Name:       "realm",
				Partitions: 1,
			}, {
				Name:       "realm",
				Partitions: 1,
			}}),
		)
	})
})
