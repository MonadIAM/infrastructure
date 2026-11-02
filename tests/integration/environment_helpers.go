//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

type savedCluster struct {
	Name         string `json:"name"`
	NamespaceUID string `json:"namespaceUID"`
}

func (fixture *clusterFixture) open(action string) {
	ginkgo.GinkgoHelper()

	directory := os.Getenv("INTEGRATION_STATE_DIR")
	gomega.Expect(filepath.IsAbs(directory)).To(gomega.BeTrue(), "INTEGRATION_STATE_DIR must be an absolute path")
	fixture.directory = filepath.Clean(directory)
	source, err := filepath.Abs("../..")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	relative, err := filepath.Rel(fixture.directory, source)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator))).To(gomega.BeTrue(), "test state must not contain the source repository")

	_, err = os.Stat(fixture.directory)
	if os.IsNotExist(err) && action == "prepare" {
		gomega.Expect(os.MkdirAll(filepath.Dir(fixture.directory), 0700)).To(gomega.Succeed())
		gomega.Expect(os.Mkdir(fixture.directory, 0700)).To(gomega.Succeed())
		fixture.name = fmt.Sprintf("infra-test-%d", time.Now().UnixNano())
		fixture.save(savedCluster{Name: fixture.name})
		fixture.prepareEnvironment(source)
	} else {
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "test environment is missing; run make itest-prepare explicitly")
	}

	data, err := os.ReadFile(filepath.Join(fixture.directory, "fixture.json"))
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "directory is not an owned integration fixture")
	var saved savedCluster
	gomega.Expect(json.Unmarshal(data, &saved)).To(gomega.Succeed())
	gomega.Expect(regexp.MustCompile(`^infra-test-[0-9]+$`).MatchString(saved.Name)).To(gomega.BeTrue())
	fixture.name = saved.Name

	lockPath := filepath.Join(fixture.directory, "run.lock")
	lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "another operation owns the test environment; inspect run.lock")
	gomega.Expect(lock.Close()).To(gomega.Succeed())
	ginkgo.DeferCleanup(func() {
		err := os.Remove(lockPath)
		if !os.IsNotExist(err) {
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		}
	})

	values, err := godotenv.Read(filepath.Join(fixture.directory, "fixture.env"))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(values["CLUSTER"]).To(gomega.Equal(fixture.name))
	gomega.Expect(values["KUBE_CONTEXT"]).To(gomega.Equal("k3d-" + fixture.name))
	gomega.Expect(values["KUBECONFIG"]).To(gomega.Equal(filepath.Join(fixture.directory, "kubeconfig")))
	fixture.environment = os.Environ()
	for key, value := range values {
		fixture.environment = append(fixture.environment, key+"="+value)
	}
}

func (fixture *clusterFixture) save(saved savedCluster) {
	ginkgo.GinkgoHelper()
	data, err := json.MarshalIndent(saved, "", "  ")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(os.WriteFile(filepath.Join(fixture.directory, "fixture.json"), data, 0600)).To(gomega.Succeed())
}

func (fixture *clusterFixture) recordCluster(ctx context.Context) {
	ginkgo.GinkgoHelper()
	identity, err := fixture.kubectl(ctx, "get", "namespace", "kube-system", "-o", "jsonpath={.metadata.uid}")
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), identity)
	gomega.Expect(identity).NotTo(gomega.BeEmpty())
	fixture.save(savedCluster{Name: fixture.name, NamespaceUID: identity})
}

func (fixture *clusterFixture) attach(ctx context.Context) {
	ginkgo.GinkgoHelper()
	data, err := os.ReadFile(filepath.Join(fixture.directory, "fixture.json"))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	var saved savedCluster
	gomega.Expect(json.Unmarshal(data, &saved)).To(gomega.Succeed())
	gomega.Expect(saved.NamespaceUID).NotTo(gomega.BeEmpty(), "preparation did not finish; run make itest-prepare explicitly")
	identity, err := fixture.kubectl(ctx, "get", "namespace", "kube-system", "-o", "jsonpath={.metadata.uid}")
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "saved test cluster is unavailable: "+identity)
	gomega.Expect(identity).To(gomega.Equal(saved.NamespaceUID), "the test cluster was replaced")
}

func (fixture *clusterFixture) prepareGitOps(ctx context.Context) {
	ginkgo.GinkgoHelper()
	ginkgo.By("Installing Argo CD and Image Updater through Terraform")
	output, err := fixture.runLogged(ctx, "make", "-f", "terraform/Makefile", "ROOT=gitops", "apply")
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)

	ginkgo.By("Preparing External Secrets and Reloader through Argo CD")
	output, err = fixture.kubectl(ctx, "apply", "--server-side",
		"-f", "clusters/dev/gitops/projects.yaml",
		"-f", "clusters/dev/gitops/external-secrets.yaml",
		"-f", "clusters/dev/gitops/reloader.yaml")
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
	for _, application := range []string{"monadiam-external-secrets", "monadiam-reloader"} {
		output, err = fixture.kubectl(ctx, "wait", "application/"+application, "-n", "argocd",
			"--for=jsonpath={.status.health.status}=Healthy", "--timeout=300s")
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
	}

	output, err = fixture.kubectl(ctx, "apply", "--server-side", "-k", "infrastructure/security/application-secrets")
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
	output, err = fixture.kubectl(ctx, "wait", "secretstore/kubernetes-data", "-n", "monadiam-apps",
		"--for=condition=Ready", "--timeout=60s")
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
}
