package seaweedfs

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/minio/minio-go/v7"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
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

	Describe("[Method] - Service.Credentials", func() {
		It("[case] - escapes credentials and restricts archiver actions to archive objects", func() {
			// Arrange
			settings := testSettings()
			settings.ArchiverSecret = "test\"\\\n${literal}"
			var actual credentialFile

			// Act
			data, err := New(settings).Credentials()
			decodeErr := json.Unmarshal(data, &actual)

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(decodeErr).To(Succeed())
			Expect(actual.Identities).To(HaveLen(2))
			Expect(actual.Identities).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":        Equal("archiver"),
				"Credentials": ConsistOf(credential{AccessKey: settings.ArchiverKey, SecretKey: settings.ArchiverSecret}),
				"Actions":     ConsistOf("Read:audit-archive/*", "Write:audit-archive/*", "Read:change-log-archive/*", "Write:change-log-archive/*"),
			})))
		})

		It("[case] - rejects shared access keys", func() {
			// Arrange
			settings := testSettings()
			settings.ArchiverKey = settings.AdminKey

			// Act
			_, err := New(settings).Credentials()

			// Assert
			Expect(err).To(MatchError("distinct admin and archiver credentials are required"))
		})
	})

	Describe("[Method] - Bucket.Validate", func() {
		DescribeTable("[Behavior] - rejects invalid bucket settings", func(name string, days int) {
			// Arrange
			bucket := Bucket{Name: name, Days: days}

			// Act
			err := bucket.Validate()

			// Assert
			Expect(err).To(HaveOccurred())
		},
			Entry("[case] - empty name", "", 1),
			Entry("[case] - path separator", "audit/logs", 1),
			Entry("[case] - IP address", "127.0.0.1", 1),
			Entry("[case] - zero retention", "audit-archive", 0),
		)
	})

	Describe("[Method] - Settings.Validate", func() {
		DescribeTable("[Behavior] - rejects invalid settings", func(change func(*Settings)) {
			// Arrange
			settings := testSettings()
			change(&settings)

			// Act
			err := settings.Validate()

			// Assert
			Expect(err).To(HaveOccurred())
		},
			Entry("[case] - missing admin key", func(settings *Settings) { settings.AdminKey = "" }),
			Entry("[case] - endpoint credentials", func(settings *Settings) { settings.Endpoint = "http://user:secret@storage" }),
			Entry("[case] - endpoint path", func(settings *Settings) { settings.Endpoint = "http://storage/bucket" }),
			Entry("[case] - unsupported protocol", func(settings *Settings) { settings.Endpoint = "ftp://storage" }),
			Entry("[case] - duplicate buckets", func(settings *Settings) { settings.Buckets[1].Name = settings.Buckets[0].Name }),
			Entry("[case] - missing bucket", func(settings *Settings) { settings.Buckets = settings.Buckets[:1] }),
		)
	})

	Describe("[Method] - Service.Reconcile", func() {
		Context("[Behavior] - configures archive retention", func() {
			It("[case] - creates locked buckets and applies lifecycle and archiver policy", func() {
				// Arrange
				fixture := &s3Fixture{}
				server := httptest.NewServer(http.HandlerFunc(fixture.serve))
				defer server.Close()
				settings := testSettings()
				settings.Endpoint = server.URL

				// Act
				err := New(settings).Reconcile(context.Background(), io.Discard)
				requests := fixture.snapshot()

				// Assert
				Expect(err).To(Succeed())
				Expect(requests).To(ContainElement(MatchFields(IgnoreExtras, Fields{"Method": Equal("PUT"), "Lock": Equal("true")})))
				Expect(requests).To(ContainElement(MatchFields(IgnoreExtras, Fields{"Query": ContainSubstring("versioning"), "Body": ContainSubstring("Enabled")})))
				Expect(requests).To(ContainElement(MatchFields(IgnoreExtras, Fields{"Query": ContainSubstring("object-lock"), "Body": ContainSubstring("<Days>1095</Days>")})))
				Expect(requests).To(ContainElement(MatchFields(IgnoreExtras, Fields{"Query": ContainSubstring("lifecycle"), "Body": ContainSubstring("<Days>365</Days>")})))
				Expect(requests).To(ContainElement(MatchFields(IgnoreExtras, Fields{"Query": ContainSubstring("policy"), "Body": ContainSubstring("ArchiverCannotDelete")})))
			})

			It("[case] - accepts already owned buckets", func() {
				// Arrange
				fixture := &s3Fixture{existing: true}
				server := httptest.NewServer(http.HandlerFunc(fixture.serve))
				defer server.Close()
				settings := testSettings()
				settings.Endpoint = server.URL

				// Act
				err := New(settings).Reconcile(context.Background(), io.Discard)

				// Assert
				Expect(err).To(Succeed())
			})
		})

		Context("[Behavior] - fails safely", func() {
			It("[case] - rejects unexpected server retention", func() {
				// Arrange
				fixture := &s3Fixture{lockDays: 1}
				server := httptest.NewServer(http.HandlerFunc(fixture.serve))
				defer server.Close()
				settings := testSettings()
				settings.Endpoint = server.URL

				// Act
				err := New(settings).Reconcile(context.Background(), io.Discard)

				// Assert
				Expect(err).To(HaveOccurred())
			})

			It("[case] - stops on authorization failure without exposing server details", func() {
				// Arrange
				fixture := &s3Fixture{status: 403}
				server := httptest.NewServer(http.HandlerFunc(fixture.serve))
				defer server.Close()
				settings := testSettings()
				settings.Endpoint = server.URL

				// Act
				err := New(settings).Reconcile(context.Background(), io.Discard)

				// Assert
				Expect(err).To(MatchError("seaweedfs.Service.waitUntilReady: S3 request failed with status 403"))
				Expect(fixture.snapshot()).To(HaveLen(1))
			})

			It("[case] - validates templates before accessing S3", func() {
				// Arrange
				fixture := &s3Fixture{}
				server := httptest.NewServer(http.HandlerFunc(fixture.serve))
				defer server.Close()
				settings := testSettings()
				settings.Endpoint = server.URL
				settings.ConfigDir = GinkgoT().TempDir()

				// Act
				err := New(settings).Reconcile(context.Background(), io.Discard)

				// Assert
				Expect(err).To(HaveOccurred())
				Expect(fixture.snapshot()).To(BeEmpty())
			})
		})
	})
})
