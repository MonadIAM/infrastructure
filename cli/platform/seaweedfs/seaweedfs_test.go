package seaweedfs

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

func TestSeaweedFS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SeaweedFS")
}

var _ = Describe("[Infrastructure] - SeaweedFS", func() {
	Describe("[Function] - Credentials", func() {
		It("[case] - escapes credentials and restricts archiver actions to archive objects", func() {
			// Arrange
			settings := testSettings()
			settings.ArchiverSecret = "test\"\\\n${literal}"
			var actual credentialFile

			// Act
			data, err := Credentials(settings)
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
			_, err := Credentials(settings)

			// Assert
			Expect(err).To(MatchError("distinct admin and archiver credentials are required"))
		})
	})

	Describe("[Function] - Validate", func() {
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

	Describe("[Function] - Reconcile", func() {
		Context("[Behavior] - configures archive retention", func() {
			It("[case] - creates locked buckets and applies lifecycle and archiver policy", func() {
				// Arrange
				fixture := &s3Fixture{}
				server := httptest.NewServer(http.HandlerFunc(fixture.serve))
				defer server.Close()
				settings := testSettings()
				settings.Endpoint = server.URL

				// Act
				err := Reconcile(context.Background(), settings, io.Discard)
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
				err := Reconcile(context.Background(), settings, io.Discard)

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
				err := Reconcile(context.Background(), settings, io.Discard)

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
				err := Reconcile(context.Background(), settings, io.Discard)

				// Assert
				Expect(err).To(MatchError("S3 request failed with status 403"))
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
				err := Reconcile(context.Background(), settings, io.Discard)

				// Assert
				Expect(err).To(HaveOccurred())
				Expect(fixture.snapshot()).To(BeEmpty())
			})
		})
	})
})
