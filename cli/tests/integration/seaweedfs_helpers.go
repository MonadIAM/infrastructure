//go:build integration

package integration

import (
	"context"
	"encoding/json"

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

	environment["SEAWEEDFS_ENDPOINT"] = "http://" + fixture.forward(ctx, "service/seaweedfs-main-s3", "8333")
	settings, err := seaweedfs.Load(environment)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return settings
}
