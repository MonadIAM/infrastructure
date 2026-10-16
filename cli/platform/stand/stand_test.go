package stand

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/command"
	"github.com/MonadIAM/infrastructure/cli/platform/helm"
)

type runnerFunc func(context.Context, command.Request) (command.Result, error)

func (run runnerFunc) Run(ctx context.Context, request command.Request) (command.Result, error) {
	return run(ctx, request)
}

func TestStand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stand")
}

var _ = Describe("[Infrastructure] - Stand", func() {
	var environment config.Environment

	BeforeEach(func() {
		var err error
		environment, err = godotenv.Read("../../../.env.example")
		Expect(err).To(Succeed())
	})

	Describe("[Method] - Platform.Start", func() {
		It("[case] - does not create a cluster when inventory fails", func() {
			// Arrange
			var requests []command.Request
			runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
				requests = append(requests, request)

				return command.Result{}, errors.New("Docker unavailable")
			})
			platform := New(environment, runner, io.Discard)

			// Act
			err := platform.Start(context.Background(), false)

			// Assert
			Expect(err).To(MatchError("Docker unavailable"))
			Expect(requests).To(HaveLen(1))
			Expect(requests[0].Args).To(Equal([]string{"cluster", "list", "-o", "json"}))
		})

		It("[case] - refuses resume when the cluster is absent", func() {
			// Arrange
			calls := 0
			runner := runnerFunc(func(_ context.Context, _ command.Request) (command.Result, error) {
				calls++

				return command.Result{Output: []byte("[]")}, nil
			})
			platform := New(environment, runner, io.Discard)

			// Act
			err := platform.Start(context.Background(), true)

			// Assert
			Expect(err).To(MatchError(ContainSubstring("does not exist")))
			Expect(calls).To(Equal(1))
		})

		It("[case] - creates the requested cluster without changing the current context", func() {
			// Arrange
			var creation command.Request
			runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
				if request.Args[0] == "cluster" && request.Args[1] == "list" {
					return command.Result{Output: []byte("[]")}, nil
				}

				creation = request

				return command.Result{}, errors.New("creation interrupted")
			})
			platform := New(environment, runner, io.Discard)

			// Act
			err := platform.Start(context.Background(), false)

			// Assert
			Expect(err).To(HaveOccurred())
			Expect(creation.Args).To(ContainElement("--kubeconfig-switch-context=false"))
			Expect(creation.Args).To(ContainElement(environment["CLUSTER"]))
		})
	})

	Describe("[Method] - Platform.Restart", func() {
		It("[case] - restarts without deleting the existing cluster", func() {
			// Arrange
			var commands []string
			runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
				commands = append(commands, request.Name+" "+strings.Join(request.Args, " "))
				if request.Name == "kubectl" {
					return command.Result{}, errors.New("readiness interrupted")
				}

				return command.Result{Output: []byte(`[{"name":"monadiam"}]`)}, nil
			})
			platform := New(environment, runner, io.Discard)

			// Act
			err := platform.Restart(context.Background())

			// Assert
			Expect(err).To(MatchError("readiness interrupted"))
			Expect(commands).To(ContainElement("k3d cluster stop monadiam"))
			Expect(commands).To(ContainElement("k3d cluster start monadiam --timeout 180s"))
			Expect(strings.Join(commands, "\n")).NotTo(ContainSubstring("cluster delete"))
		})
	})

	Describe("[Method] - Platform.KafkaResources", func() {
		It("[case] - does not apply a failed Kafka render", func() {
			// Arrange
			calls := 0
			runner := runnerFunc(func(_ context.Context, _ command.Request) (command.Result, error) {
				calls++

				return command.Result{Output: []byte("partial")}, errors.New("render failed")
			})
			platform := New(environment, runner, io.Discard)

			// Act
			err := platform.KafkaResources(context.Background())

			// Assert
			Expect(err).To(MatchError("render failed"))
			Expect(calls).To(Equal(1))
		})

		It("[case] - preserves placeholders owned by Kafka Connect and Debezium", func() {
			// Arrange
			var applied []byte
			var contexts [][]string
			runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
				contexts = append(contexts, request.Args[:2])
				if request.Args[2] == "kustomize" {
					return command.Result{Output: []byte(`${PROJECTION_JOURNAL_PARTITIONS} ${secrets:ns/password:key} ${routedByValue}`)}, nil
				}

				applied = request.Input

				return command.Result{}, nil
			})
			platform := New(environment, runner, io.Discard)

			// Act
			err := platform.KafkaResources(context.Background())

			// Assert
			Expect(err).To(Succeed())
			Expect(contexts).To(HaveEach(Equal([]string{"--context", environment["KUBE_CONTEXT"]})))
			Expect(string(applied)).To(Equal(`4 ${secrets:ns/password:key} ${routedByValue}`))
		})
	})

	Describe("[Method] - Platform.applyClusterResources", func() {
		It("[case] - establishes monitoring CRDs before applying data resources", func() {
			// Arrange
			var commands []string
			runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
				commands = append(commands, strings.Join(request.Args, " "))

				return command.Result{Output: []byte("rendered resources")}, nil
			})
			platform := New(environment, runner, io.Discard)
			platform.Helm = helm.Client{
				Runner:   runner,
				Attempts: 1,
				Output:   io.Discard,
			}

			// Act
			err := platform.applyClusterResources(context.Background())
			joined := strings.Join(commands, "\n")
			monitor := strings.Index(joined, "crd/podmonitors.monitoring.coreos.com")
			postgres := strings.Index(joined, "apply -k platform/postgresql/resources")

			// Assert
			Expect(err).To(Succeed())
			Expect(monitor).To(BeNumerically(">=", 0))
			Expect(postgres).To(BeNumerically(">", monitor))
		})
	})

	Describe("[Method] - Platform.Clusters", func() {
		It("[case] - stops when the service PostgreSQL cluster is not ready", func() {
			// Arrange
			var commands []string
			runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
				arguments := strings.Join(request.Args, " ")
				commands = append(commands, arguments)
				if strings.Contains(arguments, "cluster.postgresql.cnpg.io/template-postgresql") {
					return command.Result{}, errors.New("template PostgreSQL unavailable")
				}

				return command.Result{Output: []byte("rendered resources")}, nil
			})
			platform := New(environment, runner, io.Discard)
			platform.Helm = helm.Client{Runner: runner, Attempts: 1, Output: io.Discard}

			// Act
			err := platform.Clusters(context.Background())
			joined := strings.Join(commands, "\n")

			// Assert
			Expect(err).To(MatchError("template PostgreSQL unavailable"))
			Expect(joined).NotTo(ContainSubstring("cluster.postgresql.cnpg.io/postgresql-main"))
			Expect(commands[len(commands)-1]).To(ContainSubstring("cluster.postgresql.cnpg.io/template-postgresql"))
			for _, service := range []string{"access-control", "identity", "notification", "organization", "hr", "template"} {
				Expect(joined).To(ContainSubstring("cluster.postgresql.cnpg.io/" + service + "-postgresql"))
			}

			Expect(joined).NotTo(ContainSubstring("strimzi.io/name=kafka-main-kafka"))
		})
	})

	Describe("[Method] - Platform.Operators", func() {
		It("[case] - establishes ServiceMonitor before installing the ClickHouse operator", func() {
			// Arrange
			monitorReady := false
			operatorInstalled := false
			runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
				arguments := strings.Join(request.Args, " ")
				if strings.Contains(arguments, "crd/servicemonitors.monitoring.coreos.com") {
					monitorReady = true
				}

				if request.Name == "helm" && strings.HasPrefix(arguments, "upgrade --install clickhouse-operator ") {
					if !monitorReady {
						return command.Result{}, errors.New("ServiceMonitor is not registered")
					}

					operatorInstalled = true
				}

				return command.Result{Output: []byte("rendered resources")}, nil
			})
			platform := New(environment, runner, io.Discard)
			platform.Helm = helm.Client{
				Runner:   runner,
				Attempts: 1,
				Output:   io.Discard,
			}

			// Act
			err := platform.Operators(context.Background())

			// Assert
			Expect(err).To(Succeed())
			Expect(operatorInstalled).To(BeTrue())
		})
	})

	Describe("[Method] - Platform.MonitoringCRDs", func() {
		It("[case] - does not apply CRDs after a failed Helm download", func() {
			// Arrange
			applied := false
			runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
				if request.Name == "kubectl" {
					applied = true
				}

				if request.Args[0] == "show" {
					return command.Result{Output: []byte("partial CRD")}, errors.New("download failed")
				}

				return command.Result{}, nil
			})
			platform := New(environment, runner, io.Discard)
			platform.Helm = helm.Client{
				Runner:   runner,
				Attempts: 1,
				Output:   io.Discard,
			}

			// Act
			err := platform.MonitoringCRDs(context.Background())

			// Assert
			Expect(err).To(HaveOccurred())
			Expect(applied).To(BeFalse())
		})
	})

	Describe("[Method] - Platform.waitKafka", func() {
		DescribeTable("[Behavior] - rejects stale Kafka readiness when workloads are unavailable", func(workload string) {
			// Arrange
			runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
				if strings.Contains(strings.Join(request.Args, " "), workload) {
					return command.Result{}, errors.New("workload is unavailable")
				}

				return command.Result{}, nil
			})
			platform := New(environment, runner, io.Discard)

			// Act
			err := platform.waitKafka(context.Background())

			// Assert
			Expect(err).To(MatchError("workload is unavailable"))
		},
			Entry("[case] - broker", "strimzi.io/name=kafka-main-kafka"),
			Entry("[case] - entity operator", "deployment/kafka-main-entity-operator"),
			Entry("[case] - connect worker", "strimzi.io/name=kafka-connect-connect"),
		)
	})

	Describe("[Method] - Platform.Recreate", func() {
		It("[case] - refuses deletion without explicit confirmation", func() {
			// Arrange
			var requests []command.Request
			runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
				requests = append(requests, request)

				return command.Result{}, nil
			})
			platform := New(environment, runner, io.Discard)

			// Act
			err := platform.Recreate(context.Background(), false)

			// Assert
			Expect(err).To(MatchError(ContainSubstring("--confirm-delete")))
			Expect(requests).To(BeEmpty())
		})

		It("[case] - does not create a replacement after deletion fails", func() {
			// Arrange
			var commands []string
			runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
				arguments := strings.Join(request.Args, " ")
				commands = append(commands, arguments)
				if arguments == "cluster list -o json" {
					return command.Result{Output: []byte(`[{"name":"monadiam"}]`)}, nil
				} else {
					return command.Result{}, errors.New("deletion failed")
				}
			})
			platform := New(environment, runner, io.Discard)

			// Act
			err := platform.Recreate(context.Background(), true)

			// Assert
			Expect(err).To(MatchError("deletion failed"))
			Expect(commands).To(Equal([]string{"cluster list -o json", "cluster delete monadiam"}))
		})
	})

	Describe("[Method] - staticSecret.ready", func() {
		DescribeTable("[Behavior] - requires current and healthy synchronized secrets",
			func(generation, observed int64, mac, ready, healthy string, expected bool) {
				// Arrange
				var secret staticSecret
				secret.Metadata.Generation = generation
				secret.Status.LastGeneration = observed
				secret.Status.SecretMAC = mac
				secret.Status.Conditions = []secretCondition{
					{
						Type:   "Ready",
						Status: ready,
					},
					{
						Type:   "Healthy",
						Status: healthy,
					},
				}

				// Act
				actual := secret.ready()

				// Assert
				Expect(actual).To(Equal(expected))
			},
			Entry("[case] - synchronized generation", int64(2), int64(2), "mac", "True", "True", true),
			Entry("[case] - stale generation", int64(2), int64(1), "mac", "True", "True", false),
			Entry("[case] - missing generation", int64(0), int64(0), "mac", "True", "True", false),
			Entry("[case] - missing secret MAC", int64(2), int64(2), "", "True", "True", false),
			Entry("[case] - not ready", int64(2), int64(2), "mac", "False", "True", false),
			Entry("[case] - unhealthy", int64(2), int64(2), "mac", "True", "False", false),
		)
	})
})
