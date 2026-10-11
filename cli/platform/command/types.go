package command

import (
	"context"
	"io"

	"github.com/MonadIAM/infrastructure/cli/config"
)

type Request struct {
	Name      string
	Args      []string
	Input     []byte
	Capture   bool
	Sensitive bool
}

type Result struct {
	Output   []byte
	Stderr   []byte
	ExitCode int
}

type Runner interface {
	Run(context.Context, Request) (Result, error)
}

type Executor struct {
	Environment config.Environment
	Output      io.Writer
	Errors      io.Writer
}
