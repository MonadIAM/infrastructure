//go:build integration

package integration

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
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
	ginkgo.By("Checking tools and copying integration configuration")

	programs := []string{"go", "docker", "k3d", "kubectl", "helm", "terraform", "make"}

	for _, program := range programs {
		_, err := exec.LookPath(program)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), program)
	}

	source, err := filepath.Abs("../..")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	directories := []string{"applications", "clusters", "infrastructure", "platform", "cli"}

	for _, directory := range directories {
		gomega.Expect(os.RemoveAll(filepath.Join(fixture.directory, directory))).To(gomega.Succeed())
		err := os.CopyFS(filepath.Join(fixture.directory, directory), os.DirFS(filepath.Join(source, directory)))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	}

	files := []string{"go.mod", "go.sum", "main.go", "terraform/Makefile", "terraform/vault/config.auto.tfvars"}
	for _, pattern := range []string{"terraform/*/*.tf", "terraform/*/.terraform.lock.hcl"} {
		matches, err := filepath.Glob(filepath.Join(source, pattern))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		for _, match := range matches {
			relative, err := filepath.Rel(source, match)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			files = append(files, relative)
		}
	}

	for _, file := range files {
		gomega.Expect(os.MkdirAll(filepath.Dir(filepath.Join(fixture.directory, file)), 0700)).To(gomega.Succeed())
		data, err := os.ReadFile(filepath.Join(source, file))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(os.WriteFile(filepath.Join(fixture.directory, file), data, 0600)).To(gomega.Succeed())
	}

	fixture.prepareManifests()

	ginkgo.By("Building the infrastructure CLI")
	output, err := fixture.runLogged(ctx, "go", "build", "-o", filepath.Join(fixture.directory, "infra"), ".")
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
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
	values["GITOPS_STATE_DIR"] = filepath.Join(fixture.directory, "gitops")
	values["INFRA_TIMEOUT"] = "30m"
	values["KUBECONFIG"] = filepath.Join(fixture.directory, "kubeconfig")
	values["HELM_CACHE_HOME"] = filepath.Join(fixture.directory, "helm", "cache")
	values["HELM_CONFIG_HOME"] = filepath.Join(fixture.directory, "helm", "config")
	values["HELM_DATA_HOME"] = filepath.Join(fixture.directory, "helm", "data")
	values["HELM_REPOSITORY_CONFIG"] = filepath.Join(fixture.directory, "helm", "config", "repositories.yaml")
	values["HELM_REPOSITORY_CACHE"] = filepath.Join(fixture.directory, "helm", "cache", "repository")
	values["HELM_REGISTRY_CONFIG"] = filepath.Join(fixture.directory, "helm", "config", "registry.json")
	values["INTEGRATION_REGISTRY"] = fixture.registry()

	data, err := godotenv.Marshal(values)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(os.WriteFile(filepath.Join(fixture.directory, "fixture.env"), []byte(data), 0600)).To(gomega.Succeed())
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
	return fixture.execute(ctx, io.Discard, name, args...)
}

func (fixture *clusterFixture) runLogged(ctx context.Context, name string, args ...string) (string, error) {
	ginkgo.GinkgoHelper()
	ginkgo.By(fmt.Sprintf("%s %q", filepath.Base(name), args))
	return fixture.execute(ctx, ginkgo.GinkgoWriter, name, args...)
}

func (fixture *clusterFixture) execute(ctx context.Context, output io.Writer, name string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = fixture.directory
	command.Env = fixture.environment
	command.WaitDelay = 5 * time.Second
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		if err == syscall.ESRCH {
			return os.ErrProcessDone
		}

		return err
	}
	var captured bytes.Buffer
	command.Stdout = io.MultiWriter(&captured, output)
	command.Stderr = command.Stdout

	err := command.Run()

	return captured.String(), err
}

func (fixture *clusterFixture) cli(ctx context.Context, args ...string) (string, error) {
	return fixture.runLogged(ctx, filepath.Join(fixture.directory, "infra"), append(args, "--env-file=")...)
}

func (fixture *clusterFixture) kubectl(ctx context.Context, args ...string) (string, error) {
	options := []string{"--context", "k3d-" + fixture.name, "--request-timeout=30s"}
	return fixture.runLogged(ctx, "kubectl", append(options, args...)...)
}

func (fixture *clusterFixture) kubectlSensitive(ctx context.Context, args ...string) (string, error) {
	options := []string{"--context", "k3d-" + fixture.name, "--request-timeout=30s"}
	return fixture.run(ctx, "kubectl", append(options, args...)...)
}

func (fixture *clusterFixture) diagnostics(ctx context.Context) {
	commands := [][]string{
		{"get", "pods", "-A", "-o", "wide"},
		{"get", "pvc", "-A"},
		{"get", "events", "-A", "--field-selector=type=Warning"},
		{"logs", "-n", "vault", "vault-0", "--tail=100"},
		{"get", "kafkaconnectors.kafka.strimzi.io", "-n", "monadiam-messaging", "-o", `jsonpath={range .items[*]}{.metadata.name}{"\n"}{.status.conditions}{"\n"}{.status.connectorStatus}{"\n"}{end}`},
		{"logs", "-n", "monadiam-messaging", "-l", "strimzi.io/name=kafka-connect-connect", "--all-containers=true", "--tail=100", "--prefix=true"},
	}

	for _, args := range commands {
		_, err := fixture.kubectl(ctx, args...)
		_, _ = fmt.Fprintln(ginkgo.GinkgoWriter, strings.Join(args, " "), err)
	}

	output, err := fixture.run(ctx, "docker", "logs", "--tail", "30", "k3d-"+fixture.name+"-server-0")
	_, _ = fmt.Fprintln(ginkgo.GinkgoWriter, "k3s server logs", err)
	_, _ = io.WriteString(ginkgo.GinkgoWriter, output)
}

func (fixture *clusterFixture) cleanup(ctx context.Context) {
	ginkgo.GinkgoHelper()

	clusterOutput, clusterErr := fixture.runLogged(ctx, "k3d", "cluster", "delete", fixture.name)
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
		return fixture.runLogged(ctx, "k3d", "registry", "delete", fixture.registry())
	}
}

func (fixture *clusterFixture) forward(ctx context.Context, target, remotePort string) string {
	ginkgo.GinkgoHelper()

	if strings.HasPrefix(target, "service/") {
		service := strings.TrimPrefix(target, "service/")
		var pod string
		gomega.Eventually(func() (string, error) {
			var err error
			pod, err = fixture.endpoint(ctx, service)

			return pod, err
		}).WithContext(ctx).WithTimeout(time.Minute).ShouldNot(gomega.BeEmpty())
		target = "pod/" + pod
	}

	ctx, cancel := context.WithCancel(ctx)
	command := exec.CommandContext(ctx, "kubectl", "--context", "k3d-"+fixture.name,
		"port-forward", "--address=127.0.0.1", "-n", "monadiam-data", target, ":"+remotePort)
	ginkgo.By("Opening port-forward to " + target + ":" + remotePort)
	command.Env = fixture.environment
	command.Stderr = ginkgo.GinkgoWriter
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

	ready := make(chan portForwardResult, 1)
	go func() {
		defer close(ready)
		scanner := bufio.NewScanner(output)
		for scanner.Scan() {
			var port int
			count, err := fmt.Sscanf(scanner.Text(), "Forwarding from 127.0.0.1:%d ->", &port)
			if err == nil && count == 1 {
				ready <- portForwardResult{Port: port}
				_, _ = io.Copy(io.Discard, output)
				return
			}
		}

		ready <- portForwardResult{Err: scanner.Err()}
	}()

	var result portForwardResult
	gomega.Eventually(ready).WithContext(ctx).WithTimeout(30 * time.Second).Should(gomega.Receive(&result))
	gomega.Expect(result.Err).NotTo(gomega.HaveOccurred())
	gomega.Expect(result.Port).To(gomega.BeNumerically(">", 0), "port-forward exited before reporting a local port")

	return fmt.Sprintf("127.0.0.1:%d", result.Port)
}

func (fixture *clusterFixture) secret(ctx context.Context, name string) map[string][]byte {
	ginkgo.GinkgoHelper()
	output, err := fixture.kubectlSensitive(ctx, "get", "secret", name, "-n", "monadiam-data", "-o", "json")
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "read secret "+name)
	var secret resource
	gomega.Expect(json.Unmarshal([]byte(output), &secret)).To(gomega.Succeed())
	gomega.Expect(secret.Data).NotTo(gomega.BeEmpty(), name)
	return secret.Data
}

func (fixture *clusterFixture) endpoint(ctx context.Context, service string) (string, error) {
	output, err := fixture.kubectl(ctx, "get", "endpointslices.discovery.k8s.io", "-n", "monadiam-data", "-l", "kubernetes.io/service-name="+service,
		"-o", "jsonpath={.items[*].endpoints[?(@.conditions.ready==true)].targetRef.name}")
	if err != nil {
		return "", err
	}

	pods := strings.Fields(output)
	if len(pods) == 0 {
		return "", nil
	}

	return pods[0], nil
}
