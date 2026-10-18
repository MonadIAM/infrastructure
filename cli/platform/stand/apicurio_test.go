package stand

import (
	"context"
	"errors"
	"io"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

var _ = Describe("[Infrastructure] - Apicurio", func() {
	var expected []string

	BeforeEach(func() {
		expected = []string{
			"wait --for=condition=Ready kafka.kafka.strimzi.io/kafka-main --timeout=900s -n monadiam-messaging",
			"apply -k platform/apicurio/kafka",
			"wait --for=condition=Ready kafkatopic.kafka.strimzi.io/kafkasql-journal --timeout=300s -n monadiam-messaging",
			"wait --for=condition=Ready kafkatopic.kafka.strimzi.io/kafkasql-snapshots --timeout=300s -n monadiam-messaging",
			"wait --for=condition=Ready kafkatopic.kafka.strimzi.io/registry-events --timeout=300s -n monadiam-messaging",
			"wait --for=condition=Ready kafkauser.kafka.strimzi.io/schema-registry-main --timeout=300s -n monadiam-messaging",
			"wait --for=create secret/schema-registry-main --timeout=180s -n monadiam-messaging",
			"apply -k platform/apicurio/resources",
			"rollout status deployment/schema-registry-main -n monadiam-messaging --timeout=600s",
		}
	})

	It("[case] - runs the registry phase after cluster resources", func() {
		// Arrange
		platform := New(nil, nil, io.Discard)

		// Act
		var names []string
		for _, phase := range platform.phases() {
			names = append(names, phase.Name)
		}

		// Assert
		Expect(strings.Join(names, ",")).To(ContainSubstring("clusters,apicurio,schemas,seaweedfs"))
	})

	It("[case] - prepares Kafka storage and credentials before starting the registry", func() {
		// Arrange
		var commands []string
		runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
			Expect(request.Name).To(Equal("kubectl"))
			Expect(request.Args[:2]).To(Equal([]string{"--context", "k3d-monadiam"}))
			commands = append(commands, strings.Join(request.Args[2:], " "))

			return command.Result{}, nil
		})
		platform := New(config.Environment{"KUBE_CONTEXT": "k3d-monadiam"}, runner, io.Discard)

		// Act
		err := platform.Apicurio(context.Background())

		// Assert
		Expect(err).To(Succeed())
		Expect(commands).To(Equal(expected))
	})

	It("[case] - includes the registry in platform status", func() {
		// Arrange
		var commands []string
		runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
			commands = append(commands, request.Name+" "+strings.Join(request.Args, " "))

			return command.Result{}, nil
		})
		platform := New(config.Environment{"KUBE_CONTEXT": "k3d-monadiam"}, runner, io.Discard)

		// Act
		err := platform.Status(context.Background())

		// Assert
		Expect(err).To(Succeed())
		Expect(commands).To(ContainElement("kubectl --context k3d-monadiam get deployment,pods,svc -n monadiam-messaging -l app.kubernetes.io/name=schema-registry-main"))
	})

	DescribeTable("[case] - stops when a deployment step fails", func(failedStep int) {
		// Arrange
		failure := errors.New("deployment interrupted")
		var commands []string
		runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
			commands = append(commands, strings.Join(request.Args[2:], " "))
			if len(commands) == failedStep+1 {
				return command.Result{}, failure
			}

			return command.Result{}, nil
		})
		platform := New(config.Environment{"KUBE_CONTEXT": "k3d-monadiam"}, runner, io.Discard)

		// Act
		err := platform.Apicurio(context.Background())

		// Assert
		Expect(err).To(MatchError(failure))
		Expect(commands).To(Equal(expected[:failedStep+1]))
	},
		Entry("Kafka readiness", 0),
		Entry("Kafka resources", 1),
		Entry("journal topic", 2),
		Entry("snapshots topic", 3),
		Entry("events topic", 4),
		Entry("Kafka user", 5),
		Entry("credentials", 6),
		Entry("registry resources", 7),
		Entry("registry readiness", 8),
	)
})
