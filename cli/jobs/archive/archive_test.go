package archive

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/joho/godotenv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/MonadIAM/infrastructure/cli/platform/clickhouse"
	"github.com/MonadIAM/infrastructure/cli/platform/seaweedfs"
)

func TestArchive(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Archive job")
}

var _ = Describe("[Job] - Archive", func() {
	Describe("[Function] - archivePartition", func() {
		Context("[Behavior] - completed partitions", func() {
			It("[case] - does not revisit source rows that can expire through TTL", func() {
				// Arrange
				var requests atomic.Int32
				queries := make(chan string, 8)
				server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					requests.Add(1)
					data, err := io.ReadAll(request.Body)
					if err != nil {
						writer.WriteHeader(http.StatusBadRequest)

						return
					}

					queries <- string(data)
					_, _ = io.WriteString(writer, "1")
				}))
				DeferCleanup(server.Close)
				settings := Settings{}
				settings.ClickHouse.Address = server.URL
				client := clickhouse.Client(settings.ClickHouse, true)

				// Act
				err := archivePartition(context.Background(), client, nil, settings, table{Name: "audit_log"}, "202501")

				var captured []string
				for len(queries) > 0 {
					captured = append(captured, <-queries)
				}

				// Assert
				Expect(err).NotTo(HaveOccurred())
				Expect(requests.Load()).To(Equal(int32(1)))
				Expect(captured).To(ConsistOf(ContainSubstring("SELECT count() FROM audit.archive_log")))
			})
		})

		DescribeTable("[Behavior] - handles a Parquet object left by an interrupted cycle", func(exported string, success bool) {
			// Arrange
			environment, err := godotenv.Read("../../../.env.example")
			Expect(err).NotTo(HaveOccurred())
			settings, err := Load(environment)
			Expect(err).NotTo(HaveOccurred())
			var marks atomic.Int32
			var exports atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.Method == http.MethodHead {
					writer.Header().Set("Content-Length", "10")
					writer.Header().Set("Last-Modified", "Wed, 01 Jan 2025 00:00:00 GMT")
					writer.Header().Set("ETag", "test")

					return
				}

				data, err := io.ReadAll(request.Body)
				if err != nil {
					writer.WriteHeader(http.StatusBadRequest)

					return
				}

				query := string(data)

				switch {
				case strings.Contains(query, "INSERT INTO FUNCTION"):
					exports.Add(1)
				case strings.Contains(query, "INSERT INTO audit.archive_log"):
					marks.Add(1)
				case strings.Contains(query, "SELECT count() FROM audit.archive_log"):
					_, _ = io.WriteString(writer, "0")
				case strings.Contains(query, "SELECT count() FROM s3("):
					_, _ = io.WriteString(writer, exported)
				default:
					_, _ = io.WriteString(writer, "2")
				}
			}))
			DeferCleanup(server.Close)
			settings.ClickHouse.Address = server.URL
			settings.Storage.Endpoint = server.URL
			client := clickhouse.Client(settings.ClickHouse, true)
			storage, err := seaweedfs.Client(settings.Storage, false)
			Expect(err).NotTo(HaveOccurred())
			source := table{
				Name:       "audit_log",
				Bucket:     "audit-archive",
				Projection: "*",
			}

			// Act
			err = archivePartition(context.Background(), client, storage, settings, source, "202501")

			// Assert
			Expect(exports.Load()).To(BeZero())
			if success {
				Expect(err).NotTo(HaveOccurred())
				Expect(marks.Load()).To(Equal(int32(1)))
			} else {
				Expect(err).To(MatchError(ContainSubstring("mismatch")))
				Expect(marks.Load()).To(BeZero())
			}
		},
			Entry("[case] - verified object is recorded without reexport", "2", true),
			Entry("[case] - mismatched object is not recorded", "1", false),
		)
	})

	Describe("[Function] - Loop", func() {
		It("[case] - honors cancellation before starting a cycle", func() {
			// Arrange
			environment, err := godotenv.Read("../../../.env.example")
			Expect(err).NotTo(HaveOccurred())
			settings, err := Load(environment)
			Expect(err).NotTo(HaveOccurred())
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			// Act
			actionErr := Loop(ctx, settings, io.Discard)

			// Assert
			Expect(actionErr).To(MatchError(context.Canceled))
		})
	})
})
