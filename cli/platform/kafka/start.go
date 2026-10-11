package kafka

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/MonadIAM/infrastructure/cli/config"
)

func Start(ctx context.Context, environment config.Environment, output io.Writer) error {
	err := environment.Require("KAFKA_CLUSTER_ID")
	if err != nil {
		return err
	}

	directory := environment.Get("KAFKA_DATA_DIR", "/var/lib/kafka/data")
	configuration := environment.Get("KAFKA_CONFIG_FILE", "/etc/kafka/server.properties")
	err = os.MkdirAll(directory, 0755)
	if err != nil {
		return err
	}

	_, err = os.Stat(filepath.Join(directory, "meta.properties"))
	if errors.Is(err, os.ErrNotExist) {
		command := exec.CommandContext(ctx, "/opt/kafka/bin/kafka-storage.sh", "format", "--ignore-formatted", "--cluster-id", environment["KAFKA_CLUSTER_ID"], "--config", configuration)
		command.Stdout, command.Stderr = output, output
		err := command.Run()
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return syscall.Exec("/opt/kafka/bin/kafka-server-start.sh", []string{"/opt/kafka/bin/kafka-server-start.sh", configuration}, os.Environ())
}
