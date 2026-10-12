//go:build integration

package integration

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/joho/godotenv"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/MonadIAM/infrastructure/cli/platform/seaweedfs"
)

func (fixture *clusterFixture) storage(ctx context.Context) seaweedfs.Settings {
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

func (fixture *clusterFixture) forwardStorage(ctx context.Context) string {
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
