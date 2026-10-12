package command

import (
	"context"
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
