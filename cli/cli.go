package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/MonadIAM/infrastructure/cli/config"
)

func Run(ctx context.Context, args []string, output io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		_, err := fmt.Fprint(output, usage)

		return err
	}

	if len(args) < 2 {
		return errors.New("unknown command; use infra --help")
	}

	name := strings.Join(args[:2], " ")
	handler, exists := commands[name]
	if !exists {
		return errors.New("unknown command; use infra --help")
	}

	options, err := parseOptions(name, args[2:], output)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}

	if err != nil {
		return err
	}

	if name == "config init" {
		return handler(ctx, nil, options, output)
	}

	environment, err := config.Load(options.envFile, options.overrides)
	if err != nil {
		return err
	}

	timeout, err := environment.Duration("INFRA_TIMEOUT", "60m")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return handler(ctx, environment, options, output)
}

func parseOptions(name string, args []string, output io.Writer) (commandOptions, error) {
	options := commandOptions{overrides: make(map[string]string)}
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {
		_, _ = fmt.Fprintf(output, "Usage: infra %s [options]\n", name)
		flags.SetOutput(output)
		flags.PrintDefaults()
		flags.SetOutput(io.Discard)
	}

	flags.StringVar(&options.envFile, "env-file", ".env", "dotenv path; empty disables file loading")
	if name != "config init" {
		flags.StringVar(&options.timeout, "timeout", "", "total execution timeout, e.g. 60m")
	}

	if name == "platform reconcile" {
		flags.StringVar(&options.component, "component", "all", "platform component to reconcile")
	}

	if name == "cluster recreate" {
		flags.BoolVar(&options.confirmDelete, "confirm-delete", false, "allow deletion and recreation of the cluster")
	}

	err := flags.Parse(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return options, err
		}

		return options, errors.New("invalid options; use command --help")
	}

	if flags.NArg() != 0 {
		return options, errors.New("unexpected positional arguments")
	}

	flags.Visit(func(option *flag.Flag) {
		if option.Name == "timeout" {
			options.overrides["INFRA_TIMEOUT"] = options.timeout
		}
	})

	return options, nil
}
