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

type commandOptions struct {
	envFile      string
	csvFile      string
	arguments    []string
	connectURL   string
	templatePath string
	timeout      string
	overrides    map[string]string
}

func (options *commandOptions) applyExplicitFlags(flags *flag.FlagSet) {
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

	var environment config.Environment
	if command != "aws import" {
		environment, err = config.Load(options.envFile, options.overrides)
		if err != nil {
			return fmt.Errorf("cli.Run: %w", err)
		}
	}

	app := application{environment: environment, options: options, output: output}
	err = handler(&app, ctx)
	if err != nil {
		return fmt.Errorf("cli.Run: %s: %w", command, err)
	} else {
		return nil
	}
}

func parseOptions(command string, args []string, output io.Writer) (commandOptions, error) {
	options := commandOptions{overrides: make(map[string]string)}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}

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
			writeErr := writeHelp(command, flags, output)
			if writeErr != nil {
				return options, writeErr
			}

			return options, err
		} else {
			return options, errors.New("invalid options; use command --help")
		}
	}

	if flags.NArg() != 0 && command != "seaweedfs start" {
		return options, errors.New("unexpected positional arguments")
	}

	options.arguments = flags.Args()
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
