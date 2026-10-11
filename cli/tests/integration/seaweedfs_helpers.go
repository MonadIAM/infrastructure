//go:build integration

package integration

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/MonadIAM/infrastructure/cli/platform/seaweedfs"
)

func (fixture clusterFixture) storage(ctx context.Context) seaweedfs.Settings {
	ginkgo.GinkgoHelper()

	environment, err := godotenv.Read("../../../.env.example")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	secrets := []string{"seaweedfs-main-admin", "seaweedfs-main-archiver"}
	for _, name := range secrets {
		output, err := fixture.kubectl(ctx, "get", "secret", name, "-n", "monadiam-data", "-o", "json")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		var secret resource
		gomega.Expect(json.Unmarshal([]byte(output), &secret)).To(gomega.Succeed())
		gomega.Expect(len(secret.Data)).To(gomega.Equal(2))
		for key, value := range secret.Data {
			environment[key] = string(value)
		}
	}

	environment["SEAWEEDFS_ENDPOINT"] = fixture.forwardStorage(ctx)
	settings, err := seaweedfs.Load(environment)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return settings
}

func (fixture clusterFixture) forwardStorage(ctx context.Context) string {
	ginkgo.GinkgoHelper()

	ctx, cancel := context.WithCancel(ctx)
	command := exec.CommandContext(ctx, "kubectl", "--context", "k3d-"+fixture.name,
		"port-forward", "--address=127.0.0.1", "-n", "monadiam-data", "service/seaweedfs-main-s3", ":8333")
	command.Env = fixture.environment
	command.WaitDelay = 5 * time.Second
	output, err := command.StdoutPipe()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	err = command.Start()
	if err != nil {
		cancel()
	}
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	ginkgo.DeferCleanup(func() {
		cancel()
		_ = command.Wait()
	})

	ready := make(chan int, 1)
	go func() {
		defer close(ready)
		scanner := bufio.NewScanner(output)
		for scanner.Scan() {
			var port int
			count, err := fmt.Sscanf(scanner.Text(), "Forwarding from 127.0.0.1:%d ->", &port)
			if err == nil && count == 1 {
				ready <- port
				_, _ = io.Copy(io.Discard, output)
				return
			}
		}
	}()

	var port int
	gomega.Eventually(ready).WithContext(ctx).WithTimeout(30 * time.Second).Should(gomega.Receive(&port))
	gomega.Expect(port).To(gomega.BeNumerically(">", 0))

	return fmt.Sprintf("http://127.0.0.1:%d", port)
}

func (fixture *clusterFixture) seedArchives(ctx context.Context, settings seaweedfs.Settings) {
	ginkgo.GinkgoHelper()

	client, err := seaweedfs.Client(settings, false)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	fixture.archives = make(map[string]string)

	for _, bucket := range settings.Buckets {
		object, err := client.PutObject(ctx, bucket.Name, "integration/probe.json", strings.NewReader(fixture.name), int64(len(fixture.name)), minio.PutObjectOptions{ContentType: "application/json"})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(object.VersionID).NotTo(gomega.BeEmpty())
		fixture.archives[bucket.Name] = object.VersionID
	}
}

func (fixture clusterFixture) archivesAvailable(ctx context.Context, settings seaweedfs.Settings) error {
	client, err := seaweedfs.Client(settings, false)
	if err != nil {
		return err
	}

	for bucket, version := range fixture.archives {
		object, err := client.GetObject(ctx, bucket, "integration/probe.json", minio.GetObjectOptions{VersionID: version})
		if err != nil {
			return err
		}

		data, err := io.ReadAll(object)
		_ = object.Close()
		if err != nil {
			return err
		}

		if string(data) != fixture.name {
			return fmt.Errorf("archive content changed in %s", bucket)
		}
	}

	return nil
}

func archiveRetention(ctx context.Context, client *minio.Client, bucket seaweedfs.Bucket) error {
	enabled, mode, days, unit, err := client.GetObjectLockConfig(ctx, bucket.Name)
	if err != nil {
		return err
	}

	if enabled != "Enabled" || mode == nil || *mode != minio.Compliance || days == nil || *days != uint(bucket.Days) || unit == nil || *unit != minio.Days {
		return fmt.Errorf("unexpected retention for %s", bucket.Name)
	}

	versioning, err := client.GetBucketVersioning(ctx, bucket.Name)
	if err != nil {
		return err
	}

	if versioning.Status != "Enabled" {
		return fmt.Errorf("versioning is disabled for %s", bucket.Name)
	}

	lifecycle, err := client.GetBucketLifecycle(ctx, bucket.Name)
	if err != nil {
		return err
	}

	if len(lifecycle.Rules) != 2 || int(lifecycle.Rules[0].Expiration.Days) != bucket.Days {
		return fmt.Errorf("unexpected lifecycle for %s", bucket.Name)
	}

	return nil
}
