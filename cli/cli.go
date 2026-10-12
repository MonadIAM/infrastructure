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

type commandOptions struct {
	envFile       string
	timeout       string
	component     string
	confirmDelete bool
	overrides     map[string]string
}

func (options *commandOptions) applyExplicitFlags(flags *flag.FlagSet) {
	flags.Visit(func(option *flag.Flag) {
		if option.Name == "timeout" {
			options.overrides["INFRA_TIMEOUT"] = options.timeout
		}
	})
}

func Run(ctx context.Context, args []string, output io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		_, err := io.WriteString(output, usage)
		if err != nil {
			return fmt.Errorf("cli.Run: write help: %w", err)
		} else {
			return nil
		}
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
		app := application{options: options, output: output}

		return handler(&app, ctx)
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

	app := application{environment: environment, options: options, output: output}

	return handler(&app, ctx)
}

func parseOptions(name string, args []string, output io.Writer) (commandOptions, error) {
	options := commandOptions{overrides: make(map[string]string)}
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}

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
			writeErr := writeHelp(name, flags, output)
			if writeErr != nil {
				return options, writeErr
			}

			return options, err
		}

		return options, errors.New("invalid options; use command --help")
	}

	if flags.NArg() != 0 {
		return options, errors.New("unexpected positional arguments")
	}

	options.applyExplicitFlags(flags)

	return options, nil
}

func writeHelp(command string, flags *flag.FlagSet, output io.Writer) error {
	var defaults strings.Builder
	flags.SetOutput(&defaults)
	flags.PrintDefaults()

	help := fmt.Sprintf("Usage: infra %s [options]\n%s", command, defaults.String())
	_, err := io.WriteString(output, help)
	if err != nil {
		return fmt.Errorf("cli.writeHelp: %w", err)
	} else {
		return nil
	}
}
