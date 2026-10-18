package stand

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/joho/godotenv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/command"
	"github.com/MonadIAM/infrastructure/cli/platform/helm"
)

var _ = Describe("[Infrastructure] - Istio", func() {
	var environment config.Environment

	BeforeEach(func() {
		var err error
		environment, err = godotenv.Read("../../../.env.example")
		Expect(err).To(Succeed())
	})

	It("[case] - installs the ambient components in order before applying security policies", func() {
		var operations []string
		runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
			if request.Name == "helm" && request.Args[0] == "upgrade" {
				operations = append(operations, "install "+request.Args[2])
				Expect(request.Args).To(ContainElement("global.platform=k3d"))
				Expect(request.Args).To(ContainElement(environment["ISTIO_VERSION"]))
			}

			if request.Name == "kubectl" && request.Args[2] == "rollout" {
				operations = append(operations, "ready "+request.Args[4])
			}

			if strings.Contains(strings.Join(request.Args, " "), "infrastructure/networking/istio/resources") {
				operations = append(operations, "policies")
			}

			return command.Result{}, nil
		})
		platform := New(environment, runner, io.Discard)
		platform.Helm = helm.Client{Runner: runner, Output: io.Discard, Attempts: 1}

		err := platform.Istio(context.Background())

		Expect(err).To(Succeed())
		Expect(operations).To(Equal([]string{
			"install istio-base",
			"install istiod",
			"ready deployment/istiod",
			"install istio-cni",
			"ready daemonset/istio-cni-node",
			"install ztunnel",
			"ready daemonset/ztunnel",
			"policies",
		}))
	})

	DescribeTable("[case] - stops when an ambient component is not ready", func(workload string) {
		var commands []string
		runner := runnerFunc(func(_ context.Context, request command.Request) (command.Result, error) {
			arguments := strings.Join(request.Args, " ")
			commands = append(commands, arguments)
			if strings.Contains(arguments, "rollout status") && strings.Contains(arguments, workload) {
				return command.Result{}, errors.New("workload unavailable")
			}

			return command.Result{}, nil
		})
		platform := New(environment, runner, io.Discard)
		platform.Helm = helm.Client{Runner: runner, Output: io.Discard, Attempts: 1}

		err := platform.Istio(context.Background())

		Expect(err).To(MatchError("workload unavailable"))
		Expect(strings.Join(commands, "\n")).NotTo(ContainSubstring("infrastructure/networking/istio/resources"))
	},
		Entry("istiod", "deployment/istiod"),
		Entry("CNI", "daemonset/istio-cni-node"),
		Entry("ztunnel", "daemonset/ztunnel"),
	)
})
