package vault

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

func readCredentialPermissions(directory string) (map[string]os.FileMode, error) {
	files := []string{"root-token", "unseal-key", "init.json"}
	permissions := make(map[string]os.FileMode)

	for _, name := range files {
		info, err := os.Stat(filepath.Join(directory, name))
		if err != nil {
			return nil, err
		}

		permissions[name] = info.Mode().Perm()
	}

	return permissions, nil
}

func TestVault(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vault")
}

var _ = Describe("[Infrastructure] - Vault", func() {
	Describe("[Method] - Service.Initialize", func() {
		It("[case] - persists credentials privately and does not recreate an existing local token", func() {
			// Arrange
			var initialized atomic.Bool
			var sealed atomic.Bool
			var tokenCreated atomic.Bool
			var initializations atomic.Int32
			var creations atomic.Int32
			var unauthorized atomic.Bool
			var unexpected atomic.Bool
			sealed.Store(true)
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				switch request.URL.Path {
				case "/v1/sys/seal-status":
					_ = json.NewEncoder(writer).Encode(sealStatus{
						Initialized: initialized.Load(),
						Sealed:      sealed.Load(),
					})
				case "/v1/sys/init":
					initializations.Add(1)
					initialized.Store(true)
					_, _ = io.WriteString(writer, `{"keys":["private-key"],"root_token":"private-root"}`)
				case "/v1/sys/unseal":
					sealed.Store(false)
					_, _ = io.WriteString(writer, `{"sealed":false,"initialized":true}`)
				case "/v1/auth/token/lookup-self":
					if !tokenCreated.Load() {
						writer.WriteHeader(403)

						return
					}

					_, _ = io.WriteString(writer, `{"data":{"policies":["root"],"ttl":0}}`)
				case "/v1/auth/token/create":
					if request.Header.Get("X-Vault-Token") != "private-root" {
						unauthorized.Store(true)
					}

					creations.Add(1)
					tokenCreated.Store(true)
					_, _ = io.WriteString(writer, `{}`)
				default:
					unexpected.Store(true)
					writer.WriteHeader(http.StatusNotFound)
				}
			}))
			DeferCleanup(server.Close)
			settings := Settings{
				Address:     server.URL,
				InitDir:     GinkgoT().TempDir(),
				PoliciesDir: "unused",
				LocalToken:  "private-local",
				Timeout:     time.Second,
			}

			service := New(settings)

			// Act
			firstErr := service.Initialize(context.Background(), io.Discard)
			secondErr := service.Initialize(context.Background(), io.Discard)
			permissions, readErr := readCredentialPermissions(settings.InitDir)

			// Assert
			Expect(firstErr).To(Succeed())
			Expect(secondErr).To(Succeed())
			Expect(initializations.Load()).To(Equal(int32(1)))
			Expect(creations.Load()).To(Equal(int32(1)))
			Expect(unauthorized.Load()).To(BeFalse())
			Expect(unexpected.Load()).To(BeFalse())

			Expect(readErr).NotTo(HaveOccurred())
			Expect(permissions).To(Equal(map[string]os.FileMode{
				"root-token": 0400,
				"unseal-key": 0400,
				"init.json":  0400,
			}))
		})
	})

	Describe("[Behavior] - readiness failures", func() {
		DescribeTable("[Behavior] - stops on permanent errors without initializing Vault",
			func(code int) {
				// Arrange
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
					requests.Add(1)
					writer.WriteHeader(code)
					_, _ = io.WriteString(writer, "private-response")
				}))
				DeferCleanup(server.Close)
				settings := Settings{
					Address:     server.URL,
					InitDir:     GinkgoT().TempDir(),
					PoliciesDir: "unused",
					LocalToken:  "private-local",
					Timeout:     time.Second,
				}

				// Act
				err := New(settings).Initialize(context.Background(), io.Discard)

				// Assert
				Expect(err).To(MatchError(ContainSubstring("vault.Service.waitUntilReady: HTTP status")))
				Expect(err.Error()).NotTo(ContainSubstring("private"))
				Expect(requests.Load()).To(Equal(int32(1)))
			},
			Entry("[case] - unauthorized", http.StatusUnauthorized),
			Entry("[case] - forbidden", http.StatusForbidden),
			Entry("[case] - redirect", http.StatusTemporaryRedirect),
		)
	})

	Describe("[Behavior] - readiness response", func() {
		It("[case] - preserves a malformed JSON cause without exposing response values", func() {
			// Arrange
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(writer, `{"sealed":"private-secret"}`)
			}))
			DeferCleanup(server.Close)
			settings := Settings{
				Address:     server.URL,
				InitDir:     GinkgoT().TempDir(),
				PoliciesDir: "unused",
				LocalToken:  "private-local",
				Timeout:     time.Second,
			}

			// Act
			err := New(settings).Initialize(context.Background(), io.Discard)
			var cause *json.UnmarshalTypeError
			matched := errors.As(err, &cause)

			// Assert
			Expect(matched).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("vault.Service.waitUntilReady:"))
			Expect(err.Error()).NotTo(ContainSubstring("private-secret"))
		})

		It("[case] - cancels readiness retries without initializing Vault", func() {
			// Arrange
			ctx, cancel := context.WithCancel(context.Background())
			DeferCleanup(cancel)
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				writer.WriteHeader(http.StatusServiceUnavailable)
				cancel()
			}))
			DeferCleanup(server.Close)
			settings := Settings{
				Address:     server.URL,
				InitDir:     GinkgoT().TempDir(),
				PoliciesDir: "unused",
				LocalToken:  "private-local",
				Timeout:     time.Minute,
			}

			// Act
			err := New(settings).Initialize(ctx, io.Discard)

			// Assert
			Expect(err).To(MatchError(context.Canceled))
			Expect(requests.Load()).To(Equal(int32(1)))
		})
	})

	Describe("[Method] - Service.ensureKey", func() {
		DescribeTable("[Behavior] - creates only missing Transit keys",
			func(code int, expectedWrites int32, succeeds bool) {
				// Arrange
				var writes atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					if request.Method == http.MethodGet {
						writer.WriteHeader(code)
						_, _ = io.WriteString(writer, `{"data":{"type":"aes256-gcm96"}}`)
					} else {
						writes.Add(1)
						writer.WriteHeader(http.StatusNoContent)
					}
				}))
				DeferCleanup(server.Close)
				service := New(Settings{Address: server.URL})
				key := keyDefinition{Name: "test", Type: "aes256-gcm96"}

				// Act
				err := service.ensureKey(context.Background(), key)

				// Assert
				Expect(err == nil).To(Equal(succeeds))
				Expect(writes.Load()).To(Equal(expectedWrites))
			},
			Entry("[case] - existing key", http.StatusOK, int32(0), true),
			Entry("[case] - missing key", http.StatusNotFound, int32(1), true),
			Entry("[case] - forbidden lookup", http.StatusForbidden, int32(0), false),
			Entry("[case] - failed lookup", http.StatusServiceUnavailable, int32(0), false),
		)
	})

	Describe("[Method] - Service.prepareInit", func() {
		It("[case] - recovers a partially saved initialization without making a new init request", func() {
			// Arrange
			directory := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(directory, "init.json"), []byte(`{"keys":["key"],"root_token":"root"}`), 0400)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(directory, "root-token"), []byte("root\n"), 0400)).To(Succeed())

			// Act
			actionErr := New(Settings{Address: "http://127.0.0.1:1", InitDir: directory}).prepareInit(context.Background(), true)
			key, readErr := readSecret(directory, "unseal-key")

			// Assert
			Expect(actionErr).To(Succeed())
			Expect(readErr).NotTo(HaveOccurred())
			Expect(key).To(Equal("key"))
		})

		It("[case] - refuses to overwrite credentials when Vault storage was reset", func() {
			// Arrange
			directory := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(directory, "root-token"), []byte("saved"), 0400)).To(Succeed())

			// Act
			err := New(Settings{Address: "http://127.0.0.1:1", InitDir: directory}).prepareInit(context.Background(), false)

			// Assert
			Expect(err).To(MatchError(ContainSubstring("already exist")))
		})
	})

	Describe("[Behavior] - resource definitions", func() {
		It("[case] - keeps generated role policies and secret mappings independent between calls", func() {
			// Arrange
			roles := roleDefinitions()
			secrets := secretDefinitions()

			for _, role := range roles {
				for index := range role.Policies {
					role.Policies[index] = "changed"
				}
			}

			for _, secret := range secrets {
				for index := range secret.Fields {
					secret.Fields[index].Environment = "CHANGED"
				}
			}

			// Act
			freshRoles := roleDefinitions()
			freshSecrets := secretDefinitions()

			// Assert
			Expect(freshRoles).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":     Equal("identity-service"),
				"Policies": ContainElement("identity-service"),
			})))
			Expect(freshSecrets).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Path": Equal("identity-service/runtime"),
				"Fields": ContainElement(MatchFields(IgnoreExtras, Fields{
					"Name":        Equal("postgresql_username"),
					"Environment": Equal("IDENTITY_POSTGRES_USER"),
				})),
			})))
		})
	})

})
