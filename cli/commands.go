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

var commands = map[string]commandHandler{
	"config validate":      validateConfiguration,
	"aws import":           importAWS,
	"kafka reconcile":      reconcileKafka,
	"kafka start":          startKafka,
	"debezium reconcile":   reconcileDebezium,
	"vault initialize":     initializeVault,
	"vault bootstrap":      bootstrapVault,
	"clickhouse reconcile": reconcileClickHouse,
	"seaweedfs reconcile":  reconcileSeaweedFS,
	"seaweedfs start":      startSeaweedFS,
	"archive run":          runArchive,
	"archive loop":         loopArchive,
}

func validateConfiguration(_ context.Context, environment config.Environment, _ commandOptions, output io.Writer) error {
	settings, err := config.ParseDebezium(environment)
	if err != nil {
		return err
	}

	connectors, err := debezium.LoadConnectors(settings)
	if err != nil {
		return err
	}

	_, err = kafka.Load(environment)
	if err != nil {
		return err
	}

	_, err = schema.Load(environment)
	if err != nil {
		return err
	}

	vaultSettings, err := vault.Load(environment)
	if err != nil {
		return err
	}

	err = vault.ValidateBootstrap(vaultSettings)
	if err != nil {
		return err
	}

	storage, err := seaweedfs.Load(environment)
	if err != nil {
		return err
	}

	err = seaweedfs.ValidateTemplates(storage)
	if err != nil {
		return err
	}

	database, err := clickhouse.Load(environment)
	if err != nil {
		return err
	}

	_, err = clickhouse.Statements(database)
	if err != nil {
		return err
	}

	_, err = archive.Load(environment)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(output, "Infrastructure configuration is valid (%d connectors); Kafka, Schema Registry, Vault, ClickHouse, SeaweedFS and archiver settings checked.\n", len(connectors))
	return err
}

func importAWS(_ context.Context, _ config.Environment, options commandOptions, output io.Writer) error {
	return aws.Import(options.csvFile, options.envFile, output)
}

func reconcileKafka(ctx context.Context, environment config.Environment, _ commandOptions, output io.Writer) error {
	settings, err := kafka.Load(environment)
	if err == nil {
		return kafka.Reconcile(ctx, settings, output)
	} else {
		return err
	}
}

func startKafka(ctx context.Context, environment config.Environment, _ commandOptions, output io.Writer) error {
	return kafka.Start(ctx, environment, output)
}

func reconcileDebezium(ctx context.Context, environment config.Environment, _ commandOptions, output io.Writer) error {
	settings, err := config.ParseDebezium(environment)
	if err == nil {
		return debezium.Reconcile(ctx, settings, output)
	} else {
		return err
	}
}

func initializeVault(ctx context.Context, environment config.Environment, _ commandOptions, output io.Writer) error {
	settings, err := vault.Load(environment)
	if err == nil {
		return vault.Initialize(ctx, settings, output)
	} else {
		return err
	}
}

func bootstrapVault(ctx context.Context, environment config.Environment, _ commandOptions, output io.Writer) error {
	settings, err := vault.Load(environment)
	if err == nil {
		return vault.Bootstrap(ctx, settings, output)
	} else {
		return err
	}
}

func reconcileClickHouse(ctx context.Context, environment config.Environment, _ commandOptions, output io.Writer) error {
	settings, err := clickhouse.Load(environment)
	if err == nil {
		return clickhouse.Reconcile(ctx, settings, output)
	} else {
		return err
	}
}

func reconcileSeaweedFS(ctx context.Context, environment config.Environment, _ commandOptions, output io.Writer) error {
	settings, err := seaweedfs.Load(environment)
	if err == nil {
		return seaweedfs.Reconcile(ctx, settings, output)
	} else {
		return err
	}
}

func startSeaweedFS(_ context.Context, environment config.Environment, options commandOptions, _ io.Writer) error {
	return seaweedfs.Start(environment, options.arguments)
}

func runArchive(ctx context.Context, environment config.Environment, _ commandOptions, output io.Writer) error {
	settings, err := archive.Load(environment)
	if err == nil {
		return archive.Run(ctx, settings, output)
	} else {
		return err
	}
}

func loopArchive(ctx context.Context, environment config.Environment, _ commandOptions, output io.Writer) error {
	settings, err := archive.Load(environment)
	if err == nil {
		return archive.Loop(ctx, settings, output)
	} else {
		return err
	}
}
