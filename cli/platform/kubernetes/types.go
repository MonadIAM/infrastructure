package kubernetes

import (
	"context"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

type Client struct {
	Context string
	Runner  command.Runner
}

type resourceState struct {
	Status map[string]any `json:"status"`
}

type runnerFunc func(context.Context, command.Request) (command.Result, error)
