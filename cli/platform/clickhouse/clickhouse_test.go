package clickhouse

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestClickHouse(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ClickHouse")
}

var _ = Describe("[Infrastructure] - ClickHouse", func() {
	Describe("[Function] - Statements", func() {
		It("[case] - renders multiple schema files and escapes credentials as SQL literals", func() {
			// Arrange
			settings := Settings{
				Address:        "http://clickhouse:8123",
				User:           "default",
				Password:       "test-only",
				WriterUser:     "writer",
				WriterPassword: "secret'; DROP TABLE audit.audit_log; --\\\n\r\x00",
				ReaderUser:     "reader",
				ReaderPassword: "reader's\\secret",
				SchemaDir:      GinkgoT().TempDir(),
				Timeout:        time.Second,
			}
			Expect(os.WriteFile(filepath.Join(settings.SchemaDir, "writer.sql"), []byte("CREATE USER {{writer_user}} IDENTIFIED BY '{{writer_password}}';"), 0600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(settings.SchemaDir, "reader.sql"), []byte("CREATE USER {{reader_user}} IDENTIFIED BY '{{reader_password}}'; SELECT 'a;b';"), 0600)).To(Succeed())

			// Act
			statements, err := Statements(settings)

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(statements).To(ConsistOf(
				"CREATE USER `writer` IDENTIFIED BY "+`'secret\'; DROP TABLE audit.audit_log; --\\\n\r\0'`,
				"CREATE USER `reader` IDENTIFIED BY "+`'reader\'s\\secret'`,
				"SELECT 'a;b'",
			))
		})
	})

	Describe("[Function] - splitStatements", func() {
		It("[case] - keeps semicolons inside SQL string literals", func() {
			// Arrange
			source := "SELECT 'a;b'; SELECT 'c';"

			// Act
			statements, err := splitStatements(source)

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(statements).To(Equal([]string{"SELECT 'a;b'", "SELECT 'c'"}))
		})

		It("[case] - rejects unterminated string literals", func() {
			// Arrange
			source := "SELECT 'secret"

			// Act
			_, err := splitStatements(source)

			// Assert
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).NotTo(ContainSubstring("secret"))
		})
	})
})
