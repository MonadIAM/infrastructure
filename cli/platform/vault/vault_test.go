package vault

import (
	"context"
	"encoding/json"
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

	infrahttp "github.com/MonadIAM/infrastructure/cli/platform/http"
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
	Describe("[Function] - Initialize", func() {
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

			// Act
			firstErr := Initialize(context.Background(), settings, io.Discard)
			secondErr := Initialize(context.Background(), settings, io.Discard)
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

	Describe("[Function] - prepareInit", func() {
		It("[case] - recovers a partially saved initialization without making a new init request", func() {
			// Arrange
			directory := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(directory, "init.json"), []byte(`{"keys":["key"],"root_token":"root"}`), 0400)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(directory, "root-token"), []byte("root\n"), 0400)).To(Succeed())

			// Act
			actionErr := prepareInit(context.Background(), infrahttp.New("http://127.0.0.1:1"), Settings{InitDir: directory}, true)
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
			err := prepareInit(context.Background(), infrahttp.New("http://127.0.0.1:1"), Settings{InitDir: directory}, false)

			// Assert
			Expect(err).To(MatchError(ContainSubstring("already exist")))
		})
	})

	Describe("[Behavior] - resource definitions", func() {
		It("[case] - keeps mount options, role policies and secret mappings independent between calls", func() {
			// Arrange
			mounts := mountDefinitions()
			roles := roleDefinitions()
			secrets := secretDefinitions()
			for _, mount := range mounts {
				if mount.Options != nil {
					mount.Options["version"] = "1"
				}
			}

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
			freshMounts := mountDefinitions()
			freshRoles := roleDefinitions()
			freshSecrets := secretDefinitions()

			// Assert
			Expect(freshMounts).To(ContainElement(MatchFields(IgnoreExtras, Fields{
				"Name":    Equal("kv"),
				"Options": HaveKeyWithValue("version", "2"),
			})))
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
