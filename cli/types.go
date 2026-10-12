package cli

import (
	"context"
)

type commandHandler func(*application, context.Context) error

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
