//go:build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

type clusterFixture struct {
	name        string
	directory   string
	environment []string
}

func (fixture *clusterFixture) prepare(ctx context.Context) {
	ginkgo.GinkgoHelper()

	programs := []string{"go", "docker", "k3d", "kubectl", "helm"}

	for _, program := range programs {
		_, err := exec.LookPath(program)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), program)
	}

	source, err := filepath.Abs("../../..")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	directories := []string{"clusters", "infrastructure", "platform", "cli"}

	for _, directory := range directories {
		err := os.CopyFS(filepath.Join(fixture.directory, directory), os.DirFS(filepath.Join(source, directory)))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	}

	files := []string{"go.mod", "go.sum", "main.go"}
	for _, file := range files {
		data, err := os.ReadFile(filepath.Join(source, file))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(os.WriteFile(filepath.Join(fixture.directory, file), data, 0600)).To(gomega.Succeed())
	}

	fixture.prepareEnvironment(source)
	fixture.prepareManifests()

	command := exec.CommandContext(ctx, "go", "build", "-o", filepath.Join(fixture.directory, "infra"), ".")
	command.Dir = source
	output, err := command.CombinedOutput()
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), string(output))
}

func (fixture *clusterFixture) prepareEnvironment(source string) {
	ginkgo.GinkgoHelper()

	values, err := godotenv.Read(filepath.Join(source, ".env.example"))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	values["CLUSTER"] = fixture.name
	values["KUBE_CONTEXT"] = "k3d-" + fixture.name
	values["K3D_CONFIG"] = filepath.Join(fixture.directory, "cluster.yaml")
	values["VAULT_STATE_DIR"] = filepath.Join(fixture.directory, "vault")
	values["VAULT_INIT_FILE"] = filepath.Join(fixture.directory, "vault", "init.json")
	values["INFRA_TIMEOUT"] = "30m"
	values["KUBECONFIG"] = filepath.Join(fixture.directory, "kubeconfig")
	values["HELM_CACHE_HOME"] = filepath.Join(fixture.directory, "helm", "cache")
	values["HELM_CONFIG_HOME"] = filepath.Join(fixture.directory, "helm", "config")
	values["HELM_DATA_HOME"] = filepath.Join(fixture.directory, "helm", "data")
	values["HELM_REPOSITORY_CONFIG"] = filepath.Join(fixture.directory, "helm", "config", "repositories.yaml")
	values["HELM_REPOSITORY_CACHE"] = filepath.Join(fixture.directory, "helm", "cache", "repository")
	values["HELM_REGISTRY_CONFIG"] = filepath.Join(fixture.directory, "helm", "config", "registry.json")
	values["INTEGRATION_REGISTRY"] = fixture.registry()

	fixture.environment = os.Environ()

	for key, value := range values {
		fixture.environment = append(fixture.environment, key+"="+value)
	}
}

func (fixture *clusterFixture) prepareManifests() {
	ginkgo.GinkgoHelper()

	configuration, err := os.ReadFile("testdata/cluster.yaml")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(os.WriteFile(filepath.Join(fixture.directory, "cluster.yaml"), configuration, 0600)).To(gomega.Succeed())

	resources := filepath.Join(fixture.directory, "platform", "kafka", "resources")
	base := filepath.Join(fixture.directory, "platform", "kafka", "base")
	gomega.Expect(os.Rename(resources, base)).To(gomega.Succeed())
	gomega.Expect(os.Mkdir(resources, 0700)).To(gomega.Succeed())

	patch, err := os.ReadFile("testdata/kafka/kustomization.yaml")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	patched := strings.ReplaceAll(string(patch), "${INTEGRATION_REGISTRY}", fixture.registry())
	gomega.Expect(os.WriteFile(filepath.Join(resources, "kustomization.yaml"), []byte(patched), 0600)).To(gomega.Succeed())
}

func (fixture *clusterFixture) registry() string {
	return "k3d-" + fixture.name + "-registry"
}

func (fixture *clusterFixture) run(ctx context.Context, name string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = fixture.directory
	command.Env = fixture.environment
	command.WaitDelay = 5 * time.Second

	output, err := command.CombinedOutput()

	return string(output), err
}

func (fixture *clusterFixture) cli(ctx context.Context, args ...string) (string, error) {
	return fixture.run(ctx, filepath.Join(fixture.directory, "infra"), append(args, "--env-file=")...)
}

func (fixture *clusterFixture) kubectl(ctx context.Context, args ...string) (string, error) {
	options := []string{"--context", "k3d-" + fixture.name, "--request-timeout=30s"}

	return fixture.run(ctx, "kubectl", append(options, args...)...)
}

func (fixture *clusterFixture) diagnostics(ctx context.Context) {
	commands := [][]string{
		{"get", "pods", "-A", "-o", "wide"},
		{"get", "pvc", "-A"},
		{"get", "events", "-A", "--field-selector=type=Warning"},
	}

	for _, args := range commands {
		output, err := fixture.kubectl(ctx, args...)
		_, _ = fmt.Fprintln(ginkgo.GinkgoWriter, strings.Join(args, " "), err)
		_, _ = io.WriteString(ginkgo.GinkgoWriter, output)
	}

	output, err := fixture.run(ctx, "docker", "logs", "--tail", "30", "k3d-"+fixture.name+"-server-0")
	_, _ = fmt.Fprintln(ginkgo.GinkgoWriter, "k3s server logs", err)
	_, _ = io.WriteString(ginkgo.GinkgoWriter, output)
}

func (fixture *clusterFixture) cleanup(ctx context.Context) {
	ginkgo.GinkgoHelper()

	clusterOutput, clusterErr := fixture.run(ctx, "k3d", "cluster", "delete", fixture.name)
	registryOutput, registryErr := fixture.removeRegistry(ctx)
	containers, containersErr := fixture.run(ctx, "docker", "ps", "-a", "--filter", "name=k3d-"+fixture.name, "--format", "{{.Names}}")
	volumes, volumesErr := fixture.run(ctx, "docker", "volume", "ls", "--filter", "name=k3d-"+fixture.name, "--format", "{{.Name}}")
	networks, networksErr := fixture.run(ctx, "docker", "network", "ls", "--filter", "name=k3d-"+fixture.name, "--format", "{{.Name}}")

	gomega.Expect(clusterErr).NotTo(gomega.HaveOccurred(), clusterOutput)
	gomega.Expect(registryErr).NotTo(gomega.HaveOccurred(), registryOutput)
	gomega.Expect(containersErr).NotTo(gomega.HaveOccurred(), containers)
	gomega.Expect(volumesErr).NotTo(gomega.HaveOccurred(), volumes)
	gomega.Expect(networksErr).NotTo(gomega.HaveOccurred(), networks)
	gomega.Expect(strings.TrimSpace(containers)).To(gomega.BeEmpty())
	gomega.Expect(strings.TrimSpace(volumes)).To(gomega.BeEmpty())
	gomega.Expect(strings.TrimSpace(networks)).To(gomega.BeEmpty())
}

func (fixture *clusterFixture) removeRegistry(ctx context.Context) (string, error) {
	output, err := fixture.run(ctx, "docker", "ps", "-a", "--filter", "name=^/"+fixture.registry()+"$", "--format", "{{.Names}}")
	if err != nil {
		return output, err
	}

	if strings.TrimSpace(output) == "" {
		return "", nil
	} else {
		return fixture.run(ctx, "k3d", "registry", "delete", fixture.registry())
	}
}
