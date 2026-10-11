package vault

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"maps"

	"github.com/MonadIAM/infrastructure/cli/platform/seaweedfs"
)

func (client Client) ensureSeaweedFS(ctx context.Context) error {
	const path = "monadiam-data/seaweedfs/main/auth"

	data, version, err := client.get(ctx, path)
	if err != nil {
		return err
	}

	keys := []string{
		"SEAWEEDFS_ADMIN_ACCESS_KEY",
		"SEAWEEDFS_ADMIN_SECRET_KEY",
		"SEAWEEDFS_ARCHIVER_ACCESS_KEY",
		"SEAWEEDFS_ARCHIVER_SECRET_KEY",
	}

	if data == nil {
		data = make(map[string]string)
		for _, key := range keys {
			random := make([]byte, 32)
			_, err := rand.Read(random)
			if err != nil {
				return err
			}

			data[key] = hex.EncodeToString(random)
		}
	}

	environment := maps.Clone(client.Environment)
	for _, key := range keys {
		environment[key] = data[key]
	}

	settings, err := seaweedfs.Load(environment)
	if err != nil {
		return err
	}

	document, err := seaweedfs.Credentials(settings)
	if err != nil {
		return err
	}

	if data["seaweedfs_s3_config"] == string(document) {
		return nil
	}

	data["seaweedfs_s3_config"] = string(document)

	return client.put(ctx, path, data, fmt.Sprint(version))
}
