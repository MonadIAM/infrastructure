package stand

import (
	"context"
	"fmt"
	"io"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/command"
	"github.com/MonadIAM/infrastructure/cli/platform/helm"
	"github.com/MonadIAM/infrastructure/cli/platform/kubernetes"
	"github.com/MonadIAM/infrastructure/cli/platform/vault"
)

func New(environment config.Environment, runner command.Runner, output io.Writer) Platform {
	return Platform{
		Environment: environment,
		Runner:      runner,
		Output:      output,
		Kubernetes: kubernetes.Client{
			Context: environment["KUBE_CONTEXT"],
			Runner:  runner,
		},
	}
}

func (platform Platform) Reconcile(ctx context.Context, component string) error {
	err := config.Validate(platform.Environment)
	if err != nil {
		return err
	}

	platform.Helm, err = helm.New(platform.Environment, platform.Runner, platform.Output)
	if err != nil {
		return err
	}

	phases := platform.phases()
	if component != "all" {
		phase, exists := findPhase(phases, component)
		if !exists {
			return fmt.Errorf("unknown platform component")
		}

		phases = []phaseDefinition{phase}
	}

	for _, phase := range phases {
		err := ctx.Err()
		if err != nil {
			return err
		}

		_, _ = fmt.Fprintf(platform.Output, "Reconciling %s...\n", phase.Name)
		err = phase.Run(ctx)
		if err != nil {
			return fmt.Errorf("%s: %w", phase.Name, err)
		}
	}

	return nil
}

func findPhase(phases []phaseDefinition, name string) (phaseDefinition, bool) {
	for _, phase := range phases {
		if phase.Name == name {
			return phase, true
		}
	}

	return phaseDefinition{}, false
}

func (platform Platform) install(ctx context.Context, name string) error {
	release, exists := releaseDefinitions(platform.Environment)[name]
	if !exists {
		return fmt.Errorf("unknown Helm release %s", name)
	}

	return platform.Helm.Install(ctx, release)
}

func (platform Platform) VaultClient() vault.Client {
	return vault.Client{
		Environment: platform.Environment,
		Kubernetes:  platform.Kubernetes,
		Output:      platform.Output,
	}
}
