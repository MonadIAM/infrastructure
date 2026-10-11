package helm

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

func New(environment config.Environment, runner command.Runner, output io.Writer) (Client, error) {
	attempts, err := environment.PositiveInt("HELM_RETRY_ATTEMPTS")
	if err != nil {
		return Client{}, err
	}

	delay, err := environment.PositiveInt("HELM_RETRY_DELAY_SECONDS")
	if err != nil {
		return Client{}, err
	}

	maximum, err := environment.PositiveInt("HELM_RETRY_MAX_DELAY_SECONDS")
	if err != nil {
		return Client{}, err
	}

	return Client{
		Context:  environment["KUBE_CONTEXT"],
		Runner:   runner,
		Output:   output,
		Attempts: attempts,
		Delay:    time.Duration(delay) * time.Second,
		MaxDelay: time.Duration(maximum) * time.Second,
	}, nil
}

func (client Client) Execute(ctx context.Context, capture bool, args ...string) ([]byte, error) {
	delay := min(client.Delay, client.MaxDelay)

	for attempt := 1; attempt <= client.Attempts; attempt++ {
		err := ctx.Err()
		if err != nil {
			return nil, err
		}

		result, err := client.Runner.Run(ctx, command.Request{
			Name:    "helm",
			Args:    args,
			Capture: capture,
		})
		if err == nil {
			return result.Output, nil
		}

		contextErr := ctx.Err()
		if contextErr != nil {
			return nil, contextErr
		}

		if attempt == client.Attempts {
			return nil, err
		}

		_, _ = fmt.Fprintf(client.Output, "Helm failed; retrying in %s (%d/%d).\n", delay, attempt, client.Attempts)
		err = command.Pause(ctx, delay)
		if err != nil {
			return nil, err
		}

		delay = min(delay*2, client.MaxDelay)
	}

	return nil, fmt.Errorf("helm retry attempts must be positive")
}

func (client Client) Repo(ctx context.Context, name, address string) error {
	_, err := client.Execute(ctx, false, "repo", "add", name, address, "--force-update")

	return err
}

func (client Client) Install(ctx context.Context, release Release) error {
	args := []string{
		"upgrade", "--install", release.Name, release.Chart,
		"--version", release.Version,
		"--namespace", release.Namespace,
		"--create-namespace",
		"--kube-context", client.Context,
		"--values", release.Values,
	}
	args = append(args, release.Arguments...)

	_, err := client.Execute(ctx, false, args...)

	return err
}
