package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/command"
	"github.com/MonadIAM/infrastructure/cli/platform/seaweedfs"
	"github.com/MonadIAM/infrastructure/cli/platform/stand"
)

type application struct {
	environment config.Environment
	options     commandOptions
	output      io.Writer
}

var commands = map[string]commandHandler{
	"config init":         (*application).initializeConfiguration,
	"config validate":     (*application).validateConfiguration,
	"cluster start":       (*application).startCluster,
	"cluster resume":      (*application).resumeCluster,
	"cluster stop":        (*application).stopCluster,
	"cluster restart":     (*application).restartCluster,
	"cluster recreate":    (*application).recreateCluster,
	"cluster status":      (*application).clusterStatus,
	"platform reconcile":  (*application).reconcilePlatform,
	"vault status":        (*application).vaultStatus,
	"seaweedfs reconcile": (*application).reconcileSeaweedFS,
}

func (app *application) reconcileSeaweedFS(ctx context.Context) error {
	settings, err := seaweedfs.Load(app.environment)
	if err != nil {
		return err
	}

	return seaweedfs.New(settings).Reconcile(ctx, app.output)
}

func (app *application) initializeConfiguration(_ context.Context) error {
	err := config.Initialize(app.options.envFile)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(app.output, "Environment file ready:", app.options.envFile)

	return err
}

func (app *application) validateConfiguration(_ context.Context) error {
	err := app.environment.Validate()
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(app.output, "Stand configuration is valid.")

	return err
}

func (app *application) newPlatform() stand.Platform {
	runner := command.Executor{
		Environment: app.environment,
		Output:      app.output,
		Errors:      app.output,
	}

	return stand.New(app.environment, runner, app.output)
}

func (app *application) startCluster(ctx context.Context) error {
	return app.newPlatform().Start(ctx, false)
}

func (app *application) resumeCluster(ctx context.Context) error {
	return app.newPlatform().Start(ctx, true)
}

func (app *application) stopCluster(ctx context.Context) error {
	return app.newPlatform().Stop(ctx)
}

func (app *application) restartCluster(ctx context.Context) error {
	return app.newPlatform().Restart(ctx)
}

func (app *application) recreateCluster(ctx context.Context) error {
	return app.newPlatform().Recreate(ctx, app.options.confirmDelete)
}

func (app *application) clusterStatus(ctx context.Context) error {
	return app.newPlatform().Status(ctx)
}

func (app *application) reconcilePlatform(ctx context.Context) error {
	return app.newPlatform().Reconcile(ctx, app.options.component)
}

func (app *application) vaultStatus(ctx context.Context) error {
	client := app.newPlatform().VaultClient()
	status, err := client.Status(ctx)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(app.output)
	encoder.SetIndent("", "  ")

	return encoder.Encode(status)
}
