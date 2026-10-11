package cli

import (
	"context"
	"io"

	"github.com/MonadIAM/infrastructure/cli/config"
)

type commandHandler func(context.Context, config.Environment, commandOptions, io.Writer) error

type commandOptions struct {
	envFile      string
	csvFile      string
	arguments    []string
	connectURL   string
	templatePath string
	timeout      string
	overrides    map[string]string
}

const usage = `Usage:
  infra config validate [options]
  infra aws import [--csv=aws.csv] [--env-file=.env]
  infra kafka reconcile [options]
  infra kafka start [options]
  infra debezium reconcile [options]
  infra vault initialize [options]
  infra vault bootstrap [options]
  infra clickhouse reconcile [options]
  infra seaweedfs reconcile [options]
  infra seaweedfs start [options] -- [server arguments]
  infra archive run [options]
  infra archive loop [options]

Use --env-file= when Compose already supplies the environment.
`
