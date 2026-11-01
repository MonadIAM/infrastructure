//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func (fixture *clusterFixture) schemaSyncPatch(ctx context.Context) string {
	ginkgo.GinkgoHelper()

	// Manifests
	output, err := fixture.kubectl(ctx, "create", "--dry-run=client", "-k", "clusters/dev/applications/shared-schemas", "-o", "json")
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), output)
	decoder := json.NewDecoder(strings.NewReader(output))
	var manifests []string
	for {
		var manifest json.RawMessage
		err := decoder.Decode(&manifest)
		if errors.Is(err, io.EOF) {
			break
		}

		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		manifests = append(manifests, string(manifest))
	}

	gomega.Expect(manifests).NotTo(gomega.BeEmpty())

	// Sync operation
	operation, err := json.Marshal(map[string]any{
		"operation": map[string]any{
			"initiatedBy": map[string]string{"username": "infrastructure-integration"},
			"sync": map[string]any{
				"manifests":    manifests,
				"syncStrategy": map[string]any{"hook": map[string]any{}},
			},
		},
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	patchPath := filepath.Join(fixture.directory, "shared-schemas-sync.json")
	gomega.Expect(os.WriteFile(patchPath, operation, 0600)).To(gomega.Succeed())

	return patchPath
}
