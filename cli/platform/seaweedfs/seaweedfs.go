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

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

type Service struct {
	settings Settings
	client   *minio.Client
}

func New(settings Settings) *Service {
	return &Service{settings: settings}
}

func (service *Service) Client(admin bool) (*minio.Client, error) {
	endpoint, err := url.Parse(service.settings.Endpoint)
	if err != nil {
		return nil, errors.New("invalid SeaweedFS endpoint")
	}

	key, secret := service.settings.ArchiverKey, service.settings.ArchiverSecret
	if admin {
		key, secret = service.settings.AdminKey, service.settings.AdminSecret
	}

	return minio.New(endpoint.Host, &minio.Options{
		Creds:        credentials.NewStaticV4(key, secret, ""),
		Secure:       endpoint.Scheme == "https",
		Region:       service.settings.Region,
		BucketLookup: minio.BucketLookupPath,
	})
}

func (service *Service) ValidateTemplates() error {
	_, err := service.configurations()
	return err
}

func (service *Service) Reconcile(ctx context.Context, output io.Writer) error {
	err := service.settings.Validate()
	if err != nil {
		return err
	}

	configurations, err := service.configurations()
	if err != nil {
		return err
	}

	service.client, err = service.Client(true)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, service.settings.Timeout)
	defer cancel()

	err = service.waitUntilReady(ctx)
	if err != nil {
		return err
	}

	for _, configuration := range configurations {
		err := service.configureBucket(ctx, configuration)
		if err != nil {
			return fmt.Errorf("seaweedfs.Service.Reconcile: configure bucket %s: %w", configuration.Bucket.Name, RequestError{Cause: err})
		}
	}

	_, err = fmt.Fprintln(output, "SeaweedFS archive buckets are configured.")
	return err
}

func (service *Service) Credentials() ([]byte, error) {
	err := service.settings.Validate()
	if err != nil {
		return nil, err
	}

	if service.settings.ArchiverKey == "" || service.settings.ArchiverSecret == "" || service.settings.AdminKey == service.settings.ArchiverKey {
		return nil, errors.New("distinct admin and archiver credentials are required")
	}

	document := credentialFile{Identities: []identity{
		{
			Name: "admin",
			Credentials: []credential{{
				AccessKey: service.settings.AdminKey,
				SecretKey: service.settings.AdminSecret,
			}},
			Actions: []string{"Admin", "Read", "Write", "List", "Tagging"},
		},
		{
			Name: "archiver",
			Credentials: []credential{{
				AccessKey: service.settings.ArchiverKey,
				SecretKey: service.settings.ArchiverSecret,
			}},
			Actions: []string{
				"Read:" + service.settings.Buckets[0].Name + "/*",
				"Write:" + service.settings.Buckets[0].Name + "/*",
				"Read:" + service.settings.Buckets[1].Name + "/*",
				"Write:" + service.settings.Buckets[1].Name + "/*",
			},
		},
	}}

	return json.Marshal(document)
}

func (service *Service) configurations() ([]bucketConfiguration, error) {
	policy, err := os.ReadFile(filepath.Join(service.settings.ConfigDir, "policy.archiver.json"))
	if err != nil {
		return nil, err
	}

	rules, err := os.ReadFile(filepath.Join(service.settings.ConfigDir, "lifecycle.json"))
	if err != nil {
		return nil, err
	}

	var result []bucketConfiguration

	for _, bucket := range service.settings.Buckets {
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

func (service *Service) configureBucket(ctx context.Context, configuration bucketConfiguration) error {
	bucket := configuration.Bucket
	err := service.client.MakeBucket(ctx, bucket.Name, minio.MakeBucketOptions{
		Region:        service.settings.Region,
		ObjectLocking: true,
	})
	if err != nil && minio.ToErrorResponse(err).Code != "BucketAlreadyOwnedByYou" && minio.ToErrorResponse(err).Code != "BucketAlreadyExists" {
		return err
	}

	err = service.client.EnableVersioning(ctx, bucket.Name)
	if err != nil {
		return err
	}

	mode := minio.Compliance
	days := uint(bucket.Days)
	unit := minio.Days
	err = service.client.SetObjectLockConfig(ctx, bucket.Name, &mode, &days, &unit)
	if err != nil {
		return err
	}

	err = service.client.SetBucketPolicy(ctx, bucket.Name, configuration.Policy)
	if err != nil {
		return err
	}

	var rules lifecycle.Configuration
	err = json.Unmarshal(configuration.Lifecycle, &rules)
	if err != nil {
		return err
	}

	err = service.client.SetBucketLifecycle(ctx, bucket.Name, &rules)
	if err != nil {
		return err
	}

	enabled, actualMode, actualDays, actualUnit, err := service.client.GetObjectLockConfig(ctx, bucket.Name)
	if err != nil {
		return err
	}

	if enabled != "Enabled" || actualMode == nil || actualDays == nil || actualUnit == nil || *actualMode != mode || *actualDays != days || *actualUnit != unit {
		return errors.New("bucket retention verification failed")
	} else {
		return nil
	}
}

func (service *Service) waitUntilReady(ctx context.Context) error {
	err := command.Poll(ctx, service.settings.Timeout, time.Second, func(ctx context.Context) (bool, error) {
		_, err := service.client.ListBuckets(ctx)
		if err == nil {
			return true, nil
		}

		response := minio.ToErrorResponse(err)
		if response.StatusCode == 401 || response.StatusCode == 403 {
			return false, RequestError{Cause: err}
		}

		return false, nil
	})
	if err != nil {
		return fmt.Errorf("seaweedfs.Service.waitUntilReady: %w", err)
	} else {
		return nil
	}
}

func Load(environment config.Environment) (Settings, error) {
	settings := Settings{
		Endpoint:       environment.Get("SEAWEEDFS_ENDPOINT", "http://seaweedfs-main-s3.monadiam-data.svc:8333"),
		Region:         environment.Get("SEAWEEDFS_REGION", "us-east-1"),
		AdminKey:       environment["SEAWEEDFS_ADMIN_ACCESS_KEY"],
		AdminSecret:    environment["SEAWEEDFS_ADMIN_SECRET_KEY"],
		ArchiverKey:    environment["SEAWEEDFS_ARCHIVER_ACCESS_KEY"],
		ArchiverSecret: environment["SEAWEEDFS_ARCHIVER_SECRET_KEY"],
		ConfigDir:      environment.Get("SEAWEEDFS_CONFIG_DIR", "platform/seaweedfs/config"),
	}
	var err error
	settings.Timeout, err = environment.Duration("INFRA_TIMEOUT", "2m")
	if err != nil {
		return settings, err
	}

	bucketKinds := []string{"AUDIT_LOG", "CHANGE_LOG"}

	for _, kind := range bucketKinds {
		days, err := environment.PositiveInt("SEAWEEDFS_" + kind + "_RETENTION_DAYS")
		if err != nil {
			return settings, err
		}

		settings.Buckets = append(settings.Buckets, Bucket{
			Name: environment["SEAWEEDFS_"+kind+"_BUCKET"],
			Days: days,
		})
	}

	return settings, settings.Validate()
}
