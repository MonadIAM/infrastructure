package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/command"
	"github.com/MonadIAM/infrastructure/cli/platform/stand"
)

var commands = map[string]commandHandler{
	"config init":        initializeConfiguration,
	"config validate":    validateConfiguration,
	"cluster start":      startCluster,
	"cluster resume":     resumeCluster,
	"cluster stop":       stopCluster,
	"cluster restart":    restartCluster,
	"cluster recreate":   recreateCluster,
	"cluster status":     clusterStatus,
	"platform reconcile": reconcilePlatform,
	"vault status":       vaultStatus,
}

func initializeConfiguration(_ context.Context, _ config.Environment, options commandOptions, output io.Writer) error {
	err := config.Initialize(options.envFile)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(output, "Environment file ready:", options.envFile)

	return err
}

func validateConfiguration(_ context.Context, environment config.Environment, _ commandOptions, output io.Writer) error {
	err := config.Validate(environment)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(output, "Stand configuration is valid.")

	return err
}

func newPlatform(environment config.Environment, output io.Writer) stand.Platform {
	runner := command.Executor{
		Environment: environment,
		Output:      output,
		Errors:      output,
	}

	return stand.New(environment, runner, output)
}

func startCluster(ctx context.Context, environment config.Environment, _ commandOptions, output io.Writer) error {
	return newPlatform(environment, output).Start(ctx, false)
}

func resumeCluster(ctx context.Context, environment config.Environment, _ commandOptions, output io.Writer) error {
	return newPlatform(environment, output).Start(ctx, true)
}

func stopCluster(ctx context.Context, environment config.Environment, _ commandOptions, output io.Writer) error {
	return newPlatform(environment, output).Stop(ctx)
}

func restartCluster(ctx context.Context, environment config.Environment, _ commandOptions, output io.Writer) error {
	return newPlatform(environment, output).Restart(ctx)
}

func recreateCluster(ctx context.Context, environment config.Environment, options commandOptions, output io.Writer) error {
	return newPlatform(environment, output).Recreate(ctx, options.confirmDelete)
}

func clusterStatus(ctx context.Context, environment config.Environment, _ commandOptions, output io.Writer) error {
	return newPlatform(environment, output).Status(ctx)
}

func reconcilePlatform(ctx context.Context, environment config.Environment, options commandOptions, output io.Writer) error {
	return newPlatform(environment, output).Reconcile(ctx, options.component)
}

func vaultStatus(ctx context.Context, environment config.Environment, _ commandOptions, output io.Writer) error {
	client := newPlatform(environment, output).VaultClient()
	status, err := client.Status(ctx)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")

	return encoder.Encode(status)
}
