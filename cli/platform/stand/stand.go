package stand

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/command"
	"github.com/MonadIAM/infrastructure/cli/platform/helm"
	"github.com/MonadIAM/infrastructure/cli/platform/kubernetes"
	"github.com/MonadIAM/infrastructure/cli/platform/vault"
)

type Platform struct {
	Environment config.Environment
	Runner      command.Runner
	Output      io.Writer
	Kubernetes  kubernetes.Client
	Helm        helm.Client
}

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
	err := platform.Environment.Validate()
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
			return errors.New("unknown platform component")
		}

		phases = []phaseDefinition{phase}
		imageKeys := map[string]string{"schemas": "SHARED_SCHEMAS_IMAGE"}
		for _, application := range applications {
			imageKeys[application.Name] = application.ImageKey
		}

		if key, exists := imageKeys[component]; exists {
			phases = append([]phaseDefinition{{Name: "images", Run: func(ctx context.Context) error {
				return platform.prepareImage(ctx, key)
			}}}, phases...)
		}
	}

	for _, phase := range phases {
		err := ctx.Err()
		if err != nil {
			return err
		}

		_, _ = fmt.Fprintf(platform.Output, "Reconciling %s...\n", phase.Name)
		err = phase.Run(ctx)
		if err != nil {
			return fmt.Errorf("stand.Platform.Reconcile: %s: %w", phase.Name, err)
		}
	}

	return nil
}

func (platform Platform) VaultClient() vault.Client {
	return vault.New(platform.Environment, platform.Kubernetes, platform.Output)
}

func (platform Platform) install(ctx context.Context, name string) error {
	release, exists := releaseDefinitions(platform.Environment)[name]
	if !exists {
		return fmt.Errorf("stand.Platform.install: unknown Helm release %s", name)
	}

	return platform.Helm.Install(ctx, release)
}

func findPhase(phases []phaseDefinition, name string) (phaseDefinition, bool) {
	for _, phase := range phases {
		if phase.Name == name {
			return phase, true
		}
	}

	return phaseDefinition{}, false
}
