package vault

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
)

func (service *Service) Execute(ctx context.Context, output io.Writer, arguments ...string) error {
	if len(arguments) == 0 {
		return errors.New("command is required after --")
	}

	token, err := readSecret(service.settings.InitDir, "root-token")
	if err != nil {
		return err
	}

	command := exec.CommandContext(ctx, arguments[0], arguments[1:]...)
	command.Env = append(os.Environ(), "VAULT_ADDR="+service.settings.Address, "VAULT_TOKEN="+token)
	command.Stdout, command.Stderr = output, output
	return command.Run()
}
