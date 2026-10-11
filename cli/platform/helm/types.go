package helm

import (
	"context"
	"io"
	"time"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

type Client struct {
	Context  string
	Runner   command.Runner
	Output   io.Writer
	Attempts int
	Delay    time.Duration
	MaxDelay time.Duration
}

type Release struct {
	Name      string
	Chart     string
	Version   string
	Namespace string
	Values    string
	Arguments []string
}

type runnerFunc func(context.Context, command.Request) (command.Result, error)

type writerFunc func([]byte) (int, error)
