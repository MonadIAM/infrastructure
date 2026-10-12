package cli

import (
	"context"
)

type commandHandler func(*application, context.Context) error

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
  infra seaweedfs reconcile [options]

Options: --env-file=.env, --timeout=60m
Components: all, cert-manager, metrics-server, istio, gateway, vault,
            operators, clusters, seaweedfs, observability
Run from the stand repository root. Use --env-file= for environment-only configuration.
`
