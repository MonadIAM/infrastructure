package seaweedfs

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Settings struct {
	Endpoint       string
	Region         string
	AdminKey       string
	AdminSecret    string
	ArchiverKey    string
	ArchiverSecret string
	ConfigDir      string
	Timeout        time.Duration
	Buckets        []Bucket
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
	endpoint, err := url.Parse(settings.Endpoint)
	if err != nil || endpoint.Hostname() == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") {
		return errors.New("SeaweedFS endpoint must be an HTTP(S) URL")
	}

	if endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || (endpoint.Path != "" && endpoint.Path != "/") {
		return errors.New("SeaweedFS endpoint must not contain credentials, a path, query or fragment")
	}

	if settings.AdminKey == "" || settings.AdminSecret == "" {
		return errors.New("SeaweedFS admin credentials are required")
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
