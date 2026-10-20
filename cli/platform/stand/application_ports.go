package stand

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

func (platform Platform) publishApplicationPorts(ctx context.Context) error {
	node := "k3d-" + platform.Environment["CLUSTER"] + "-serverlb"
	result, err := platform.Runner.Run(ctx, command.Request{
		Name:    "docker",
		Args:    []string{"inspect", node, "--format", "{{json .HostConfig.PortBindings}}"},
		Capture: true,
	})
	if err != nil {
		return err
	}

	var bindings map[string][]portBinding
	err = json.Unmarshal(result.Output, &bindings)
	if err != nil {
		return fmt.Errorf("stand.Platform.publishApplicationPorts: invalid port bindings: %w", err)
	}

	args := []string{"node", "edit", node}
	for _, application := range applications {
		expected := portBinding{HostIP: platform.Environment["APPLICATIONS_IP"], HostPort: application.Port}
		current := bindings[application.Port+"/tcp"]
		if slices.Contains(current, expected) {
			continue
		}

		for _, binding := range current {
			args = append(args, "--port-delete", binding.HostIP+":"+binding.HostPort+":"+application.Port)
		}

		args = append(args, "--port-add", expected.HostIP+":"+expected.HostPort+":"+application.Port)
	}

	if len(args) > 3 {
		_, err = platform.Runner.Run(ctx, command.Request{Name: "k3d", Args: args})
	}

	return err
}
