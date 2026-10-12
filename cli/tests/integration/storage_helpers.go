//go:build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"

	"github.com/MonadIAM/infrastructure/cli/platform/seaweedfs"
)

type storageClient struct {
	client  *minio.Client
	buckets []seaweedfs.Bucket
}

func newStorageClient(settings seaweedfs.Settings, admin bool) (*storageClient, error) {
	client, err := seaweedfs.New(settings).Client(admin)
	if err != nil {
		return nil, err
	} else {
		return &storageClient{client: client, buckets: settings.Buckets}, nil
	}
}

func (storage *storageClient) seed(ctx context.Context, contents string) (map[string]string, error) {
	versions := make(map[string]string)

	for _, bucket := range storage.buckets {
		object, err := storage.client.PutObject(ctx, bucket.Name, "integration/probe.json", strings.NewReader(contents), int64(len(contents)), minio.PutObjectOptions{ContentType: "application/json"})
		if err != nil {
			return nil, fmt.Errorf("integration.storageClient.seed: %w", err)
		}

		if object.VersionID == "" {
			return nil, errors.New("storage returned an empty object version")
		}

		versions[bucket.Name] = object.VersionID
	}

	return versions, nil
}

func (storage *storageClient) read(ctx context.Context, versions map[string]string, contents string) error {
	for bucket, version := range versions {
		object, err := storage.client.GetObject(ctx, bucket, "integration/probe.json", minio.GetObjectOptions{VersionID: version})
		if err != nil {
			return fmt.Errorf("integration.storageClient.read: %w", err)
		}

		data, readErr := io.ReadAll(object)
		closeErr := object.Close()
		err = errors.Join(readErr, closeErr)
		if err != nil {
			return fmt.Errorf("integration.storageClient.read: %w", err)
		}

		if string(data) != contents {
			return fmt.Errorf("integration.storageClient.read: archive content changed in %s", bucket)
		}
	}

	return nil
}

func (storage *storageClient) retention(ctx context.Context, bucket seaweedfs.Bucket) error {
	enabled, mode, days, unit, err := storage.client.GetObjectLockConfig(ctx, bucket.Name)
	if err != nil {
		return err
	}

	if enabled != "Enabled" || mode == nil || *mode != minio.Compliance || days == nil || *days != uint(bucket.Days) || unit == nil || *unit != minio.Days {
		return fmt.Errorf("integration.storageClient.retention: unexpected retention for %s", bucket.Name)
	}

	versioning, err := storage.client.GetBucketVersioning(ctx, bucket.Name)
	if err != nil {
		return err
	}

	if versioning.Status != "Enabled" {
		return fmt.Errorf("integration.storageClient.retention: versioning is disabled for %s", bucket.Name)
	}

	lifecycle, err := storage.client.GetBucketLifecycle(ctx, bucket.Name)
	if err != nil {
		return err
	}

	if len(lifecycle.Rules) != 2 || int(lifecycle.Rules[0].Expiration.Days) != bucket.Days {
		return fmt.Errorf("integration.storageClient.retention: unexpected lifecycle for %s", bucket.Name)
	}

	return nil
}
