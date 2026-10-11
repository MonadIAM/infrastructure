package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Configuration")
}

var _ = Describe("[Config] - Environment", func() {
	Describe("[Function] - Load", func() {
		It("[case] - prefers flags to process values and process values to dotenv", func() {
			// Arrange
			path := filepath.Join(GinkgoT().TempDir(), "test.env")
			Expect(os.WriteFile(path, []byte("CLUSTER=file\nKUBE_CONTEXT=file\n"), 0600)).To(Succeed())
			GinkgoT().Setenv("CLUSTER", "process")

			// Act
			environment, err := Load(path, map[string]string{"KUBE_CONTEXT": "flag"})
			processCluster := os.Getenv("CLUSTER")

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(environment).To(HaveKeyWithValue("CLUSTER", "process"))
			Expect(environment).To(HaveKeyWithValue("KUBE_CONTEXT", "flag"))
			Expect(processCluster).To(Equal("process"))
		})

		It("[case] - treats command substitutions as data", func() {
			// Arrange
			directory := GinkgoT().TempDir()
			path := filepath.Join(directory, "test.env")
			marker := filepath.Join(directory, "executed")
			Expect(os.WriteFile(path, []byte("VALUE='$(touch "+marker+")'\n"), 0600)).To(Succeed())
			GinkgoT().Setenv("VALUE", "")
			Expect(os.Unsetenv("VALUE")).To(Succeed())

			// Act
			environment, err := Load(path, nil)
			_, statErr := os.Stat(marker)

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(environment["VALUE"]).To(ContainSubstring("$(touch"))
			Expect(os.IsNotExist(statErr)).To(BeTrue())
		})

		It("[case] - preserves an explicitly empty process value", func() {
			// Arrange
			path := filepath.Join(GinkgoT().TempDir(), "test.env")
			Expect(os.WriteFile(path, []byte("CLUSTER=file\n"), 0600)).To(Succeed())
			GinkgoT().Setenv("CLUSTER", "")

			// Act
			environment, err := Load(path, nil)

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(environment).To(HaveKeyWithValue("CLUSTER", ""))
		})

		It("[case] - does not disclose malformed dotenv contents", func() {
			// Arrange
			path := filepath.Join(GinkgoT().TempDir(), "test.env")
			Expect(os.WriteFile(path, []byte("KEY=\"private-value"), 0600)).To(Succeed())

			// Act
			_, err := Load(path, nil)

			// Assert
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).NotTo(ContainSubstring("private-value"))
		})
	})

	Describe("[Function] - Initialize", func() {
		BeforeEach(func() {
			GinkgoT().Chdir(GinkgoT().TempDir())
			Expect(os.WriteFile(".env.example", []byte("CLUSTER=example\n"), 0644)).To(Succeed())
		})

		Context("[Behavior] - environment file persistence", func() {
			It("[case] - creates a private copy of the example", func() {
				// Arrange
				path := ".env"

				// Act
				err := Initialize(path)
				info, statErr := os.Stat(path)
				content, readErr := os.ReadFile(path)

				// Assert
				Expect(err).NotTo(HaveOccurred())
				Expect(statErr).NotTo(HaveOccurred())
				Expect(readErr).NotTo(HaveOccurred())
				Expect(info.Mode().Perm()).To(Equal(os.FileMode(0600)))
				Expect(string(content)).To(Equal("CLUSTER=example\n"))
			})

			It("[case] - preserves an existing environment file", func() {
				// Arrange
				Expect(os.WriteFile(".env", []byte("CLUSTER=custom\n"), 0600)).To(Succeed())

				// Act
				err := Initialize(".env")
				content, readErr := os.ReadFile(".env")

				// Assert
				Expect(err).NotTo(HaveOccurred())
				Expect(readErr).NotTo(HaveOccurred())
				Expect(string(content)).To(Equal("CLUSTER=custom\n"))
			})
		})

		DescribeTable("[Behavior] - rejects unavailable paths", func(path string) {
			// Arrange

			// Act
			err := Initialize(path)

			// Assert
			Expect(err).To(HaveOccurred())
		},
			Entry("[case] - empty path", ""),
			Entry("[case] - missing parent directory", "missing/.env"),
		)
	})

	Describe("[Function] - Validate", func() {
		DescribeTable("[Behavior] - rejects invalid environment settings", func(key, value, message string) {
			// Arrange
			values, readErr := godotenv.Read("../../.env.example")
			Expect(readErr).NotTo(HaveOccurred())
			environment := Environment(values)
			environment[key] = value

			// Act
			err := Validate(environment)

			// Assert
			Expect(err).To(MatchError(ContainSubstring(message)))
		},
			Entry("[case] - missing cluster name", "CLUSTER", "", "CLUSTER is required"),
			Entry("[case] - zero retry attempts", "HELM_RETRY_ATTEMPTS", "0", "HELM_RETRY_ATTEMPTS"),
			Entry("[case] - malformed partition count", "PROJECTION_JOURNAL_PARTITIONS", "private-value", "PROJECTION_JOURNAL_PARTITIONS"),
			Entry("[case] - unseal threshold exceeds shares", "VAULT_KEY_THRESHOLD", "2", "VAULT_KEY_THRESHOLD must not exceed VAULT_KEY_SHARES"),
			Entry("[case] - invalid deadline", "INFRA_TIMEOUT", "0s", "INFRA_TIMEOUT"),
		)
	})
})
