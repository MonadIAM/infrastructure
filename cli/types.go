package cli

import (
	"context"
	"io"

	"github.com/MonadIAM/infrastructure/cli/config"
)

type commandHandler func(context.Context, config.Environment, commandOptions, io.Writer) error

type commandOptions struct {
	envFile       string
	timeout       string
	component     string
	confirmDelete bool
	overrides     map[string]string
}

const usage = `Usage:
  infra config init [--env-file=.env]
  infra config validate [options]
  infra cluster start [options]
  infra cluster resume [options]
  infra cluster stop [options]
  infra cluster restart [options]
  infra cluster recreate --confirm-delete [options]
  infra cluster status [options]
  infra platform reconcile [--component=all] [options]
  infra vault status [options]

Options: --env-file=.env, --timeout=60m
Components: all, cert-manager, metrics-server, istio, gateway, vault,
            operators, clusters, observability
Run from the stand repository root. Use --env-file= for environment-only configuration.
`
