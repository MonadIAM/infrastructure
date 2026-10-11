package seaweedfs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

func TestSeaweedFS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SeaweedFS")
}

var _ = Describe("[Infrastructure] - SeaweedFS", func() {
	Describe("[Function] - WriteCredentials", func() {
		It("[case] - escapes credentials and limits archiver actions to the configured buckets", func() {
			// Arrange
			environment, err := godotenv.Read("../../../.env.example")
			Expect(err).NotTo(HaveOccurred())
			settings, err := Load(environment)
			Expect(err).NotTo(HaveOccurred())
			settings.ArchiverSecret = "test\"\\\n${literal}"
			path := filepath.Join(GinkgoT().TempDir(), "credentials.json")
			var actual credentialFile

			// Act
			actionErr := WriteCredentials(settings, path)
			data, readErr := os.ReadFile(path)
			decodeErr := json.Unmarshal(data, &actual)
			info, statErr := os.Stat(path)

			// Assert
			Expect(actionErr).To(Succeed())
			Expect(readErr).NotTo(HaveOccurred())
			Expect(decodeErr).To(Succeed())
			Expect(statErr).NotTo(HaveOccurred())
			Expect(actual.Identities).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name": Equal("archiver"),
				"Credentials": ConsistOf(credential{
					AccessKey: settings.ArchiverKey,
					SecretKey: settings.ArchiverSecret,
				}),
				"Actions": ConsistOf("Read:audit-archive/*", "Write:audit-archive/*", "Read:change-log-archive/*", "Write:change-log-archive/*"),
			})))
			Expect(info.Mode().Perm()).To(Equal(os.FileMode(0600)))
		})
	})

	Describe("[Method] - Bucket.Validate", func() {
		DescribeTable("[Behavior] - rejects invalid bucket settings", func(name string, days int) {
			// Arrange
			bucket := Bucket{
				Name: name,
				Days: days,
			}

			// Act
			err := bucket.Validate()

			// Assert
			Expect(err).NotTo(Succeed())
		},
			Entry("[case] - empty name", "", 1),
			Entry("[case] - path separator", "audit/logs", 1),
			Entry("[case] - IP address", "127.0.0.1", 1),
			Entry("[case] - zero retention", "audit-archive", 0),
		)
	})
})
