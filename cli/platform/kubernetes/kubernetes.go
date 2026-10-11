package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

func (client Client) Execute(ctx context.Context, request command.Request) (command.Result, error) {
	request.Name = "kubectl"
	request.Args = append([]string{"--context", client.Context}, request.Args...)

	return client.Runner.Run(ctx, request)
}

func (client Client) Run(ctx context.Context, args ...string) error {
	_, err := client.Execute(ctx, command.Request{Args: args})

	return err
}

func (client Client) Capture(ctx context.Context, args ...string) ([]byte, error) {
	result, err := client.Execute(ctx, command.Request{
		Args:    args,
		Capture: true,
	})

	return result.Output, err
}

func (client Client) Apply(ctx context.Context, data []byte, serverSide bool) error {
	args := []string{"apply", "-f", "-"}

	if serverSide {
		args = append(args, "--server-side")
	}

	_, err := client.Execute(ctx, command.Request{
		Args:  args,
		Input: data,
	})

	return err
}

func (client Client) ApplyKustomize(ctx context.Context, path string) error {
	return client.Run(ctx, "apply", "-k", path)
}

func (client Client) Wait(ctx context.Context, namespace, resource, condition, timeout string) error {
	args := []string{"wait", "--for=" + condition, resource, "--timeout=" + timeout}

	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	return client.Run(ctx, args...)
}

func (client Client) Rollout(ctx context.Context, namespace, resource, timeout string) error {
	return client.Run(ctx, "rollout", "status", resource, "-n", namespace, "--timeout="+timeout)
}

func (client Client) WaitPods(ctx context.Context, namespace, selector, timeout string) error {
	conditions := []string{"create", "condition=Ready"}

	for _, condition := range conditions {
		err := client.Run(ctx, "wait", "--for="+condition, "pod", "-n", namespace, "-l", selector, "--timeout="+timeout)
		if err != nil {
			return err
		}
	}

	return nil
}

func (client Client) Get(ctx context.Context, namespace, resource string, result any) error {
	data, err := client.Capture(ctx, "get", resource, "-n", namespace, "-o", "json", "--request-timeout=10s")
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, result)
	if err != nil {
		return fmt.Errorf("invalid JSON for %s/%s", namespace, resource)
	}

	return nil
}

func (client Client) WaitState(ctx context.Context, namespace, resource, field, expected string, timeout time.Duration) error {
	err := command.Poll(ctx, timeout, 5*time.Second, func(ctx context.Context) (bool, error) {
		var current resourceState

		err := client.Get(ctx, namespace, resource, &current)
		if err == nil {
			return current.Status[field] == expected, nil
		} else {
			return false, err
		}
	})
	if err != nil {
		return fmt.Errorf("waiting for %s/%s: %w", namespace, resource, err)
	}

	return nil
}
