package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/jobs/archive"
	"github.com/MonadIAM/infrastructure/cli/platform/aws"
	"github.com/MonadIAM/infrastructure/cli/platform/clickhouse"
	"github.com/MonadIAM/infrastructure/cli/platform/debezium"
	"github.com/MonadIAM/infrastructure/cli/platform/kafka"
	"github.com/MonadIAM/infrastructure/cli/platform/schema"
	"github.com/MonadIAM/infrastructure/cli/platform/seaweedfs"
	"github.com/MonadIAM/infrastructure/cli/platform/vault"
)

type application struct {
	environment config.Environment
	options     commandOptions
	output      io.Writer
}

var commands = map[string]commandHandler{
	"config validate":      (*application).validateConfiguration,
	"aws import":           (*application).importAWS,
	"kafka reconcile":      (*application).reconcileKafka,
	"kafka start":          (*application).startKafka,
	"debezium reconcile":   (*application).reconcileDebezium,
	"vault initialize":     (*application).initializeVault,
	"vault bootstrap":      (*application).bootstrapVault,
	"clickhouse reconcile": (*application).reconcileClickHouse,
	"seaweedfs reconcile":  (*application).reconcileSeaweedFS,
	"seaweedfs start":      (*application).startSeaweedFS,
	"archive run":          (*application).runArchive,
	"archive loop":         (*application).loopArchive,
}

func (app *application) validateConfiguration(_ context.Context) error {
	settings, err := app.environment.ParseDebezium()
	if err != nil {
		return err
	}

	connectors, err := debezium.New(settings).LoadConnectors()
	if err != nil {
		return err
	}

	_, err = kafka.Load(app.environment)
	if err != nil {
		return err
	}

	_, err = schema.Load(app.environment)
	if err != nil {
		return err
	}

	vaultSettings, err := vault.Load(app.environment)
	if err != nil {
		return err
	}

	err = vault.New(vaultSettings).ValidateBootstrap()
	if err != nil {
		return err
	}

	storage, err := seaweedfs.Load(app.environment)
	if err != nil {
		return err
	}

	err = seaweedfs.New(storage).ValidateTemplates()
	if err != nil {
		return err
	}

	database, err := clickhouse.Load(app.environment)
	if err != nil {
		return err
	}

	_, err = clickhouse.New(database, false).Statements()
	if err != nil {
		return err
	}

	_, err = archive.Load(app.environment)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(app.output, "Infrastructure configuration is valid (%d connectors); Kafka, Schema Registry, Vault, ClickHouse, SeaweedFS and archiver settings checked.\n", len(connectors))
	return err
}

func (app *application) importAWS(_ context.Context) error {
	return aws.New(app.options.csvFile, app.options.envFile).Run(app.output)
}

func (app *application) reconcileKafka(ctx context.Context) error {
	settings, err := kafka.Load(app.environment)
	if err == nil {
		return kafka.New(settings).Reconcile(ctx, app.output)
	} else {
		return err
	}
}

func (app *application) startKafka(ctx context.Context) error {
	return kafka.Start(ctx, app.environment, app.output)
}

func (app *application) reconcileDebezium(ctx context.Context) error {
	settings, err := app.environment.ParseDebezium()
	if err == nil {
		return debezium.New(settings).Reconcile(ctx, app.output)
	} else {
		return err
	}
}

func (app *application) initializeVault(ctx context.Context) error {
	settings, err := vault.Load(app.environment)
	if err == nil {
		return vault.New(settings).Initialize(ctx, app.output)
	} else {
		return err
	}
}

func (app *application) bootstrapVault(ctx context.Context) error {
	settings, err := vault.Load(app.environment)
	if err == nil {
		return vault.New(settings).Bootstrap(ctx, app.output)
	} else {
		return err
	}
}

func (app *application) reconcileClickHouse(ctx context.Context) error {
	settings, err := clickhouse.Load(app.environment)
	if err == nil {
		return clickhouse.New(settings, false).Reconcile(ctx, app.output)
	} else {
		return err
	}
}

func (app *application) reconcileSeaweedFS(ctx context.Context) error {
	settings, err := seaweedfs.Load(app.environment)
	if err == nil {
		return seaweedfs.New(settings).Reconcile(ctx, app.output)
	} else {
		return err
	}
}

func (app *application) startSeaweedFS(_ context.Context) error {
	settings, err := seaweedfs.Load(app.environment)
	if err == nil {
		return seaweedfs.New(settings).Start(app.options.arguments)
	} else {
		return err
	}
}

func (app *application) runArchive(ctx context.Context) error {
	settings, err := archive.Load(app.environment)
	if err == nil {
		return archive.New(settings, app.output).Run(ctx)
	} else {
		return err
	}
}

func (app *application) loopArchive(ctx context.Context) error {
	settings, err := archive.Load(app.environment)
	if err == nil {
		return archive.New(settings, app.output).Loop(ctx)
	} else {
		return err
	}
}
