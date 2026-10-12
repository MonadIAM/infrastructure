package clickhouse

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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
	Describe("[Method] - Client.Query", func() {
		It("[case] - sends SQL and escaped parameter values separately in the POST body", func() {
			// Arrange
			requests := make(chan *http.Request, 1)
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				err := request.ParseMultipartForm(1 << 20)
				requests <- request
				if err != nil {
					writer.WriteHeader(http.StatusBadRequest)
					return
				}

				_, _ = io.WriteString(writer, "7\n")
			}))
			DeferCleanup(server.Close)
			client := New(Settings{Address: server.URL, Timeout: time.Second}, false)
			query := "SELECT {value:String}, {count:UInt64}"
			value := "private'; SELECT 1; --\t\n\r\x00\\N"
			parameters := map[string]string{"value": value, "count": "7"}

			// Act
			result, err := client.Query(context.Background(), query, parameters)
			request := <-requests

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("7"))
			Expect(request.Method).To(Equal(http.MethodPost))
			Expect(request.FormValue("query")).To(Equal(query))
			Expect(request.FormValue("param_value")).To(Equal(`private'; SELECT 1; --\t\n\r\0\\N`))
			Expect(request.FormValue("param_count")).To(Equal("7"))
			Expect(request.URL.Query()).NotTo(HaveKey("param_value"))
			Expect(request.URL.Query()).NotTo(HaveKey("query"))
			Expect(request.URL.Query().Get("wait_end_of_query")).To(Equal("1"))
			Expect(request.URL.Query().Get("log_queries")).To(Equal("0"))
			Expect(parameters["value"]).To(Equal(value))
		})

		It("[case] - reports the status without exposing rejected parameter values", func() {
			// Arrange
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(writer, "cannot parse private-value")
			}))
			DeferCleanup(server.Close)
			client := New(Settings{Address: server.URL, Timeout: time.Second}, false)
			parameters := map[string]string{"count": "private-value"}

			// Act
			_, err := client.Query(context.Background(), "SELECT {count:UInt64}", parameters)

			// Assert
			Expect(err).To(MatchError("clickhouse.Client.Query: HTTP status 400"))
		})
	})

	Describe("[Method] - Client.Statements", func() {
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
			statements, err := New(settings, false).Statements()

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
