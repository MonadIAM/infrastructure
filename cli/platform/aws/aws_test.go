package aws

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAWS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AWS import")
}

var _ = Describe("[Utility] - AWS credentials", func() {
	Describe("[Method] - Importer.Run", func() {
		It("[case] - reads quoted CRLF CSV and preserves unrelated environment lines", func() {
			// Arrange
			directory := GinkgoT().TempDir()
			csv := filepath.Join(directory, "aws.csv")
			env := filepath.Join(directory, ".env")
			Expect(os.WriteFile(csv, []byte("Access key ID,Secret access key\r\nTESTKEY,\"test/secret+=\"\r\n"), 0600)).To(Succeed())
			Expect(os.WriteFile(env, []byte("UNRELATED=keep\nNOTIFICATION_AWS_ACCESS_KEY_ID=old\nNOTIFICATION_AWS_SECRET_ACCESS_KEY=old\n"), 0600)).To(Succeed())
			var output bytes.Buffer

			// Act
			actionErr := New(csv, env).Run(&output)
			data, readErr := os.ReadFile(env)
			info, statErr := os.Stat(env)

			// Assert
			Expect(actionErr).To(Succeed())
			Expect(readErr).NotTo(HaveOccurred())
			Expect(statErr).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring("UNRELATED=keep\n"))
			Expect(string(data)).To(ContainSubstring("NOTIFICATION_AWS_ACCESS_KEY_ID='TESTKEY'"))
			Expect(string(data)).NotTo(ContainSubstring("=old"))
			Expect(string(data)).To(ContainSubstring("NOTIFICATION_AWS_SECRET_ACCESS_KEY='test/secret+='"))
			Expect(output.String()).NotTo(ContainSubstring("test/secret"))
			Expect(info.Mode().Perm()).To(Equal(os.FileMode(0600)))
		})

		It("[case] - leaves the environment untouched for malformed CSV", func() {
			// Arrange
			directory := GinkgoT().TempDir()
			csv := filepath.Join(directory, "aws.csv")
			env := filepath.Join(directory, ".env")
			Expect(os.WriteFile(csv, []byte("wrong,columns\na,b\n"), 0600)).To(Succeed())
			Expect(os.WriteFile(env, []byte("UNCHANGED=yes"), 0600)).To(Succeed())

			// Act
			actionErr := New(csv, env).Run(&bytes.Buffer{})
			data, readErr := os.ReadFile(env)

			// Assert
			Expect(actionErr).NotTo(Succeed())
			Expect(readErr).NotTo(HaveOccurred())
			Expect(string(data)).To(Equal("UNCHANGED=yes"))
		})
	})
})
