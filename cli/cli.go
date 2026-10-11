package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
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

	command := strings.Join(args[:2], " ")
	handler, exists := commands[command]
	if !exists {
		return errors.New("unknown command; use infra --help")
	}

	options, err := parseOptions(command, args[2:], output)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}

	if err != nil {
		return err
	}

	if command == "aws import" {
		return handler(ctx, nil, options, output)
	}

	environment, err := config.Load(options.envFile, options.overrides)
	if err == nil {
		return handler(ctx, environment, options, output)
	} else {
		return err
	}
}

func parseOptions(command string, args []string, output io.Writer) (commandOptions, error) {
	options := commandOptions{overrides: make(map[string]string)}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {
		_, _ = fmt.Fprintf(output, "Usage: infra %s [options]\n", command)
		flags.SetOutput(output)
		flags.PrintDefaults()
		flags.SetOutput(io.Discard)
	}

	flags.StringVar(&options.envFile, "env-file", ".env", "dotenv path; empty disables file loading")
	flags.StringVar(&options.connectURL, "connect-url", "", "Kafka Connect URL (overrides CONNECT_URL)")
	flags.StringVar(&options.templatePath, "template", "", "connector template path (overrides CONNECTOR_TEMPLATE)")
	flags.StringVar(&options.timeout, "timeout", "", "total execution timeout, e.g. 2m")
	if command == "aws import" {
		flags.StringVar(&options.csvFile, "csv", "aws.csv", "AWS credentials CSV path")
	}

	err := flags.Parse(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return options, err
		} else {
			return options, errors.New("invalid options; use command --help")
		}
	}

	if flags.NArg() != 0 && command != "seaweedfs start" {
		return options, errors.New("unexpected positional arguments")
	}

	options.arguments = flags.Args()
	applyExplicitFlags(flags, &options)

	return options, nil
}

func applyExplicitFlags(flags *flag.FlagSet, options *commandOptions) {
	explicitEnvFile := false

	flags.Visit(func(option *flag.Flag) {
		switch option.Name {
		case "env-file":
			explicitEnvFile = true
		case "connect-url":
			options.overrides["CONNECT_URL"] = options.connectURL
		case "template":
			options.overrides["CONNECTOR_TEMPLATE"] = options.templatePath
		case "timeout":
			options.overrides["DEBEZIUM_TIMEOUT"] = options.timeout
			options.overrides["INFRA_TIMEOUT"] = options.timeout
			options.overrides["ARCHIVE_TIMEOUT"] = options.timeout
		}
	})

	if explicitEnvFile {
		return
	}

	_, err := os.Stat(options.envFile)
	if errors.Is(err, os.ErrNotExist) {
		options.envFile = ""
	}
}
