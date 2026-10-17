//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func (fixture *clusterFixture) pods(ctx context.Context, selector string) ([]resource, error) {
	output, err := fixture.kubectl(ctx, "get", "pods", "-n", "monadiam-data", "-l", selector, "-o", "json")
	if err != nil {
		return nil, err
	}

	var pods resourceList
	err = json.Unmarshal([]byte(output), &pods)

	return pods.Items, err
}

func (fixture *clusterFixture) interruptPod(ctx context.Context, pod, selector string, count int) func(context.Context) {
	ginkgo.GinkgoHelper()
	output, err := fixture.kubectl(ctx, "get", "pod", pod, "-n", "monadiam-data", "-o", "jsonpath={.spec.nodeName}")
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
	node := strings.TrimSpace(output)
	gomega.Expect(node).NotTo(gomega.BeEmpty())
	output, err = fixture.kubectl(ctx, "get", "node", node, "-o", "jsonpath={.spec.unschedulable}")
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
	wasCordoned := strings.TrimSpace(output) == "true"
	restore := func(ctx context.Context) {
		ginkgo.GinkgoHelper()
		if !wasCordoned {
			output, err := fixture.kubectl(ctx, "uncordon", node)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
		}

		fixture.waitPods(ctx, selector, count)
	}
	ginkgo.DeferCleanup(restore, ginkgo.NodeTimeout(6*time.Minute))
	output, err = fixture.kubectl(ctx, "cordon", node)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
	output, err = fixture.kubectl(ctx, "delete", "pod", pod, "-n", "monadiam-data", "--grace-period=0", "--force", "--wait=true", "--timeout=30s")
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)

	return restore
}

func (fixture *clusterFixture) waitPods(ctx context.Context, selector string, count int) {
	ginkgo.GinkgoHelper()
	gomega.Eventually(func() (int, error) {
		pods, err := fixture.pods(ctx, selector)
		if err != nil {
			return 0, err
		}

		ready := 0
		for _, pod := range pods {
			for _, condition := range pod.Status.Conditions {
				if condition.Type == "Ready" && condition.Status == "True" {
					ready++
				}
			}
		}

		return ready, nil
	}).WithContext(ctx).WithTimeout(5 * time.Minute).WithPolling(2 * time.Second).Should(gomega.Equal(count))
}
