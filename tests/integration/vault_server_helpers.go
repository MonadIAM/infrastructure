//go:build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/vault"
)

type vaultServerFixture struct {
	root        string
	name        string
	directory   string
	environment config.Environment
}

func newVaultServerFixture() vaultServerFixture {
	ginkgo.GinkgoHelper()
	root, err := filepath.Abs("../..")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	port := listener.Addr().(*net.TCPAddr).Port
	gomega.Expect(listener.Close()).To(gomega.Succeed())
	name := fmt.Sprintf("vault-test-%d", time.Now().UnixNano())
	directory := ginkgo.GinkgoT().TempDir()
	environment, err := config.Load(filepath.Join(root, ".env.example"), map[string]string{
		"VAULT_CONTAINER_NAME":   name,
		"COMPOSE_PROJECT_NAME":   name,
		"DOCKER_NETWORK_VAULT":   name,
		"VAULT_PUBLISHED_PORT":   fmt.Sprint(port),
		"VAULT_SERVER_STATE_DIR": filepath.Join(directory, "terraform"),
		"VAULT_ADDR":             fmt.Sprintf("http://127.0.0.1:%d", port),
		"VAULT_INIT_DIR":         filepath.Join(directory, "init"),
		"VAULT_TOKEN":            "test-local-token",
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return vaultServerFixture{root: root, name: name, directory: directory, environment: environment}
}

func (fixture vaultServerFixture) terraform(ctx context.Context, action string) (string, error) {
	command := exec.CommandContext(ctx, "make", "-f", "terraform/Makefile", "ROOT=vault-server", action)
	command.Dir = fixture.root
	command.Env = os.Environ()
	for key, value := range fixture.environment.StringMap("") {
		command.Env = append(command.Env, key+"="+value)
	}

	command.Env = append(command.Env, "TF_CLI_ARGS_plan=-detailed-exitcode")
	output, err := command.CombinedOutput()
	return string(output), err
}

func (fixture vaultServerFixture) initialize(ctx context.Context) error {
	settings, err := vault.Load(fixture.environment)
	if err != nil {
		return err
	}

	return vault.New(settings).Initialize(ctx, io.Discard)
}

func (fixture vaultServerFixture) docker(ctx context.Context, arguments ...string) (string, error) {
	command := exec.CommandContext(ctx, "docker", append([]string{"--context", fixture.environment.StringMap("")["DOCKER_CONTEXT"]}, arguments...)...)
	output, err := command.CombinedOutput()
	return string(output), err
}
