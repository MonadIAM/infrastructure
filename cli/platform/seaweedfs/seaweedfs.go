package seaweedfs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"

	infrahttp "github.com/MonadIAM/infrastructure/cli/platform/http"
)

func Client(settings Settings, admin bool) (*minio.Client, error) {
	endpoint, err := url.Parse(settings.Endpoint)
	if err != nil {
		return nil, errors.New("invalid SeaweedFS endpoint")
	}

	key, secret := settings.ArchiverKey, settings.ArchiverSecret
	if admin {
		key, secret = settings.AdminKey, settings.AdminSecret
	}

	return minio.New(endpoint.Host, &minio.Options{
		Creds:        credentials.NewStaticV4(key, secret, ""),
		Secure:       endpoint.Scheme == "https",
		Region:       settings.Region,
		BucketLookup: minio.BucketLookupPath,
	})
}

func ValidateTemplates(settings Settings) error {
	_, err := configurations(settings)
	return err
}

func Reconcile(ctx context.Context, settings Settings, output io.Writer) error {
	err := settings.Validate()
	if err != nil {
		return err
	}

	configurations, err := configurations(settings)
	if err != nil {
		return err
	}

	client, err := Client(settings, true)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, settings.Timeout)
	defer cancel()

	err = infrahttp.Wait(ctx, time.Second, func() error {
		_, err := client.ListBuckets(ctx)
		return safeError(err)
	})
	if err != nil {
		return err
	}

	for _, configuration := range configurations {
		err := configureBucket(ctx, client, settings.Region, configuration)
		if err != nil {
			return fmt.Errorf("configure bucket %s: %w", configuration.Bucket.Name, safeError(err))
		}
	}

	_, err = fmt.Fprintln(output, "SeaweedFS archive buckets are configured.")
	return err
}

func configurations(settings Settings) ([]bucketConfiguration, error) {
	policy, err := os.ReadFile(filepath.Join(settings.ConfigDir, "policy.archiver.json"))
	if err != nil {
		return nil, err
	}

	rules, err := os.ReadFile(filepath.Join(settings.ConfigDir, "lifecycle.json"))
	if err != nil {
		return nil, err
	}

	var result []bucketConfiguration

	for _, bucket := range settings.Buckets {
		rendered := strings.ReplaceAll(string(policy), "BUCKET_NAME", bucket.Name)
		expiry := strings.ReplaceAll(string(rules), `"RETENTION_DAYS"`, strconv.Itoa(bucket.Days))
		if !json.Valid([]byte(rendered)) || !json.Valid([]byte(expiry)) {
			return nil, errors.New("invalid archive bucket configuration template")
		}

		result = append(result, bucketConfiguration{
			Bucket:    bucket,
			Policy:    rendered,
			Lifecycle: []byte(expiry),
		})
	}

	return result, nil
}

func configureBucket(ctx context.Context, client *minio.Client, region string, configuration bucketConfiguration) error {
	bucket := configuration.Bucket
	err := client.MakeBucket(ctx, bucket.Name, minio.MakeBucketOptions{
		Region:        region,
		ObjectLocking: true,
	})
	if err != nil && minio.ToErrorResponse(err).Code != "BucketAlreadyOwnedByYou" && minio.ToErrorResponse(err).Code != "BucketAlreadyExists" {
		return err
	}

	err = client.EnableVersioning(ctx, bucket.Name)
	if err != nil {
		return err
	}

	mode := minio.Compliance
	days := uint(bucket.Days)
	unit := minio.Days
	err = client.SetObjectLockConfig(ctx, bucket.Name, &mode, &days, &unit)
	if err != nil {
		return err
	}

	err = client.SetBucketPolicy(ctx, bucket.Name, configuration.Policy)
	if err != nil {
		return err
	}

	var rules lifecycle.Configuration
	err = json.Unmarshal(configuration.Lifecycle, &rules)
	if err != nil {
		return err
	}

	err = client.SetBucketLifecycle(ctx, bucket.Name, &rules)
	if err != nil {
		return err
	}

	enabled, actualMode, actualDays, actualUnit, err := client.GetObjectLockConfig(ctx, bucket.Name)
	if err != nil {
		return err
	}

	if enabled != "Enabled" || actualMode == nil || actualDays == nil || actualUnit == nil || *actualMode != mode || *actualDays != days || *actualUnit != unit {
		return errors.New("bucket retention verification failed")
	} else {
		return nil
	}
}

func safeError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	response := minio.ToErrorResponse(err)
	if response.StatusCode != 0 {
		return infrahttp.StatusError{Code: response.StatusCode}
	} else {
		return errors.New("S3 request failed")
	}
}

func WriteCredentials(settings Settings, path string) error {
	err := settings.Validate()
	if err != nil {
		return err
	}

	document := credentialFile{Identities: []identity{
		{
			Name: "admin",
			Credentials: []credential{{
				AccessKey: settings.AdminKey,
				SecretKey: settings.AdminSecret,
			}},
			Actions: []string{"Admin", "Read", "Write", "List", "Tagging"},
		},
		{
			Name: "archiver",
			Credentials: []credential{{
				AccessKey: settings.ArchiverKey,
				SecretKey: settings.ArchiverSecret,
			}},
			Actions: []string{
				"Read:" + settings.Buckets[0].Name + "/*",
				"Write:" + settings.Buckets[0].Name + "/*",
				"Read:" + settings.Buckets[1].Name + "/*",
				"Write:" + settings.Buckets[1].Name + "/*",
			},
		},
	}}
	data, err := json.Marshal(document)
	if err != nil {
		return err
	}

	temporary, err := os.CreateTemp(filepath.Dir(path), ".credentials-*")
	if err != nil {
		return err
	}

	defer func() { _ = os.Remove(temporary.Name()) }()

	_, err = temporary.Write(data)
	if err != nil {
		_ = temporary.Close()
		return err
	}

	err = temporary.Close()
	if err == nil {
		return os.Rename(temporary.Name(), path)
	} else {
		return err
	}
}
