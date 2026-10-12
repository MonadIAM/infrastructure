package seaweedfs

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

type Settings struct {
	Endpoint       string        `koanf:"SEAWEEDFS_ENDPOINT"`
	Region         string        `koanf:"SEAWEEDFS_REGION"`
	AdminKey       string        `koanf:"SEAWEEDFS_ADMIN_ACCESS_KEY"`
	AdminSecret    string        `koanf:"SEAWEEDFS_ADMIN_SECRET_KEY"`
	ArchiverKey    string        `koanf:"SEAWEEDFS_ARCHIVER_ACCESS_KEY"`
	ArchiverSecret string        `koanf:"SEAWEEDFS_ARCHIVER_SECRET_KEY"`
	ConfigDir      string        `koanf:"SEAWEEDFS_CONFIG_DIR"`
	Timeout        time.Duration `koanf:"INFRA_TIMEOUT"`
	Buckets        []Bucket      `koanf:"-"`
}

type Bucket struct {
	Name string
	Days int
}

type credential struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

type identity struct {
	Name        string       `json:"name"`
	Credentials []credential `json:"credentials"`
	Actions     []string     `json:"actions"`
}

type credentialFile struct {
	Identities []identity `json:"identities"`
}

type bucketConfiguration struct {
	Bucket    Bucket
	Policy    string
	Lifecycle []byte
}

func (settings Settings) Validate() error {
	if settings.Endpoint == "" {
		return errors.New("SeaweedFS endpoint is required")
	}

	if settings.AdminKey == "" || settings.AdminSecret == "" || settings.ArchiverKey == "" || settings.ArchiverSecret == "" {
		return errors.New("SeaweedFS admin and archiver credentials are required")
	}

	if settings.AdminKey == settings.ArchiverKey {
		return errors.New("admin and archiver must use different access keys")
	}

	if settings.Region == "" || settings.ConfigDir == "" || settings.Timeout <= 0 {
		return errors.New("SeaweedFS region, config path and positive timeout are required")
	}

	if len(settings.Buckets) != 2 {
		return errors.New("two archive buckets are required")
	}

	if settings.Buckets[0].Name == settings.Buckets[1].Name {
		return errors.New("archive buckets must be distinct")
	}

	for _, bucket := range settings.Buckets {
		err := bucket.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

func (bucket Bucket) Validate() error {
	valid := regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)
	if !valid.MatchString(bucket.Name) || strings.Contains(bucket.Name, "..") || net.ParseIP(bucket.Name) != nil {
		return errors.New("invalid archive bucket name")
	}

	if bucket.Days <= 0 {
		return fmt.Errorf("seaweedfs.Bucket.Validate: %s retention must be positive", bucket.Name)
	} else {
		return nil
	}
}
