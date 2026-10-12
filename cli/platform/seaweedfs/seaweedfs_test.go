package seaweedfs

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	"github.com/MonadIAM/infrastructure/cli/config"
)

func TestSeaweedFS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SeaweedFS")
}

var _ = Describe("[Infrastructure] - SeaweedFS", func() {
	Describe("[Method] - RequestError.Error", func() {
		DescribeTable("[Behavior] - hides remote details and preserves the cause", func(cause error, message string) {
			// Arrange
			failure := RequestError{Cause: cause}

			// Act
			actual := failure.Error()
			unwrapped := errors.Is(failure, cause)

			// Assert
			Expect(actual).To(Equal(message))
			Expect(actual).NotTo(ContainSubstring("private-secret"))
			Expect(unwrapped).To(BeTrue())
		},
			Entry("[case] - S3 authorization failure", minio.ErrorResponse{StatusCode: 403, Message: "private-secret"}, "S3 request failed with status 403"),
			Entry("[case] - request cancellation", &url.Error{URL: "http://host/private-secret", Err: context.Canceled}, "context canceled"),
			Entry("[case] - transport failure", &url.Error{URL: "http://host/private-secret", Err: errors.New("private-secret")}, "S3 request failed"),
		)
	})

	Describe("[Method] - Service.WriteCredentials", func() {
		It("[case] - escapes credentials and limits archiver actions to the configured buckets", func() {
			// Arrange
			environment, err := config.ReadFile("../../../.env.example")
			Expect(err).NotTo(HaveOccurred())
			settings, err := Load(environment)
			Expect(err).NotTo(HaveOccurred())
			settings.ArchiverSecret = "test\"\\\n${literal}"
			path := filepath.Join(GinkgoT().TempDir(), "credentials.json")
			var actual credentialFile

			// Act
			actionErr := New(settings).WriteCredentials(path)
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

	Describe("[Method] - Service.waitUntilReady", func() {
		It("[case] - stops immediately when S3 rejects credentials", func() {
			// Arrange
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				writer.WriteHeader(http.StatusForbidden)
			}))
			DeferCleanup(server.Close)
			service := New(Settings{Endpoint: server.URL, Region: "us-east-1"})
			client, err := service.Client(true)
			service.client = client
			Expect(err).NotTo(HaveOccurred())
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			DeferCleanup(cancel)

			// Act
			err = service.waitUntilReady(ctx)

			// Assert
			Expect(err).To(MatchError("seaweedfs.Service.waitUntilReady: S3 request failed with status 403"))
			Expect(requests.Load()).To(Equal(int32(1)))
		})

		It("[case] - cancels readiness retries", func() {
			// Arrange
			ctx, cancel := context.WithCancel(context.Background())
			DeferCleanup(cancel)
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(http.StatusServiceUnavailable)
				cancel()
			}))
			DeferCleanup(server.Close)
			service := New(Settings{Endpoint: server.URL, Region: "us-east-1"})
			client, err := service.Client(true)
			service.client = client
			Expect(err).NotTo(HaveOccurred())

			// Act
			err = service.waitUntilReady(ctx)

			// Assert
			Expect(err).To(MatchError(context.Canceled))
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
