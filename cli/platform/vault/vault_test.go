package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/command"
	"github.com/MonadIAM/infrastructure/cli/platform/kubernetes"
)

func (run runnerFunc) Run(ctx context.Context, request command.Request) (command.Result, error) {
	return run(ctx, request)
}

func readBackups(path string) (map[string]string, error) {
	paths, err := filepath.Glob(path + ".*.bak")
	if err != nil {
		return nil, err
	}

	contents := make(map[string]string)

	for _, backup := range paths {
		data, err := os.ReadFile(backup)
		if err != nil {
			return nil, err
		}

		contents[backup] = string(data)
	}

	return contents, nil
}

func TestVault(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vault")
}

var _ = Describe("[Infrastructure] - Vault", func() {
	var client Client
	var requests []command.Request
	var run runnerFunc
	var credential credentialDefinition

	BeforeEach(func() {
		requests = nil
		path := filepath.Join(GinkgoT().TempDir(), "vault", "init.json")
		Expect(os.MkdirAll(filepath.Dir(path), 0700)).To(Succeed())
		Expect(os.WriteFile(path, []byte(`{"root_token":"private-token","unseal_keys_b64":["first-key","second-key"]}`), 0600)).To(Succeed())

		credential = credentialDefinition{
			Path:        "monadiam-data/postgresql/main/superuser",
			UserKey:     "username",
			User:        "postgres",
			PasswordKey: "password",
		}
		client = Client{
			Environment: config.Environment{
				"KUBE_CONTEXT":        "stand",
				"VAULT_NAMESPACE":     "vault",
				"VAULT_POD":           "vault-0",
				"VAULT_ADDR":          "https://vault.vault.svc:8200",
				"VAULT_CACERT":        "/vault/ca.crt",
				"VAULT_STATE_DIR":     filepath.Dir(path),
				"VAULT_INIT_FILE":     path,
				"VAULT_KV_MOUNT":      "kv",
				"VAULT_KEY_SHARES":    "2",
				"VAULT_KEY_THRESHOLD": "2",
			},
			Output: io.Discard,
			Kubernetes: kubernetes.Client{
				Context: "stand",
				Runner: runnerFunc(func(ctx context.Context, request command.Request) (command.Result, error) {
					requests = append(requests, request)

					return run(ctx, request)
				}),
			},
		}
	})

	Describe("[Function] - ensureCredential", func() {
		It("[case] - preserves existing credentials", func() {
			// Arrange
			run = func(_ context.Context, _ command.Request) (command.Result, error) {
				return command.Result{Output: []byte(`{"data":{"data":{"password":"existing"},"metadata":{"version":3}}}`)}, nil
			}

			// Act
			err := client.ensureCredential(context.Background(), credential)

			// Assert
			Expect(err).To(Succeed())
			Expect(requests).To(HaveLen(1))
			Expect(requests[0].Args).To(ContainElement("get"))
		})

		It("[case] - propagates read failures without replacing credentials", func() {
			// Arrange
			run = func(_ context.Context, _ command.Request) (command.Result, error) {
				return command.Result{Stderr: []byte("permission denied private-token")}, errors.New("Vault rejected request")
			}

			// Act
			err := client.ensureCredential(context.Background(), credential)

			// Assert
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).NotTo(ContainSubstring("private-token"))
			Expect(requests).To(HaveLen(1))
		})

		It("[case] - creates missing secrets with CAS and JSON on stdin", func() {
			// Arrange
			var write command.Request
			run = func(_ context.Context, request command.Request) (command.Result, error) {
				if slices.Contains(request.Args, "get") {
					return command.Result{Stderr: []byte("No value found at kv/data/path")}, errors.New("not found")
				}

				write = request

				return command.Result{}, nil
			}

			// Act
			err := client.ensureCredential(context.Background(), credential)
			token, payload, separated := bytes.Cut(write.Input, []byte("\n"))
			var fields map[string]string
			decodeErr := json.Unmarshal(payload, &fields)
			arguments := strings.Join(write.Args, " ")

			// Assert
			Expect(err).To(Succeed())
			Expect(requests).To(HaveLen(2))
			Expect(requests).To(HaveEach(HaveField("Sensitive", BeTrue())))
			Expect(write.Args).To(ContainElement("-cas=0"))
			Expect(separated).To(BeTrue())
			Expect(string(token)).To(Equal("private-token"))
			Expect(decodeErr).To(Succeed())
			Expect(fields["username"]).To(Equal("postgres"))
			Expect(fields["password"]).To(HaveLen(64))
			Expect(arguments).NotTo(ContainSubstring(fields["password"]))
			Expect(arguments).NotTo(ContainSubstring("private-token"))
		})
	})

	Describe("[Function] - Unseal", func() {
		Context("[Behavior] - submits keys and verifies server state", func() {
			It("[case] - uses the server threshold and keeps keys out of arguments", func() {
				// Arrange
				var keys []string
				var arguments []string
				client.Environment["VAULT_KEY_THRESHOLD"] = "1"
				run = func(_ context.Context, request command.Request) (command.Result, error) {
					arguments = append(arguments, strings.Join(request.Args, " "))
					if slices.Contains(request.Args, "status") {
						if len(keys) == 2 {
							return command.Result{Output: []byte(`{"initialized":true,"sealed":false,"t":2}`)}, nil
						} else {
							return command.Result{
								ExitCode: 2,
								Output:   []byte(`{"initialized":true,"sealed":true,"t":2}`),
							}, errors.New("sealed")
						}
					}

					var input map[string]string
					err := json.Unmarshal(request.Input, &input)
					if err != nil {
						return command.Result{}, err
					}

					keys = append(keys, input["key"])

					return command.Result{}, nil
				}

				// Act
				err := client.Unseal(context.Background())

				// Assert
				Expect(err).To(Succeed())
				Expect(keys).To(Equal([]string{"first-key", "second-key"}))
				Expect(arguments).To(HaveEach(Not(ContainSubstring("first-key"))))
				Expect(arguments).To(HaveEach(Not(ContainSubstring("second-key"))))
				Expect(requests).To(HaveEach(HaveField("Sensitive", BeTrue())))
			})

			It("[case] - fails if Vault remains sealed after accepting keys", func() {
				// Arrange
				run = func(_ context.Context, request command.Request) (command.Result, error) {
					if slices.Contains(request.Args, "status") {
						return command.Result{Output: []byte(`{"initialized":true,"sealed":true,"t":2}`)}, nil
					} else {
						return command.Result{}, nil
					}
				}

				// Act
				err := client.Unseal(context.Background())

				// Assert
				Expect(err).To(MatchError("vault remains sealed after submitting the key threshold"))
			})
		})

		DescribeTable("[Behavior] - stops before submitting unavailable keys", func(contents, message string) {
			// Arrange
			Expect(os.WriteFile(client.Environment["VAULT_INIT_FILE"], []byte(contents), 0600)).To(Succeed())
			run = func(_ context.Context, _ command.Request) (command.Result, error) {
				return command.Result{Output: []byte(`{"initialized":true,"sealed":true,"t":3}`)}, nil
			}

			// Act
			err := client.Unseal(context.Background())

			// Assert
			Expect(err).To(MatchError(message))
			Expect(requests).To(HaveLen(1))
		},
			Entry("[case] - invalid init material", `{"root_token":"private-token"}`, "invalid Vault init material"),
			Entry("[case] - insufficient shares", `{"root_token":"private-token","unseal_keys_b64":["first-key"]}`, "insufficient Vault unseal keys"),
		)
	})

	Describe("[Function] - Initialize", func() {
		It("[case] - backs up old material and saves the new response privately", func() {
			// Arrange
			run = func(_ context.Context, request command.Request) (command.Result, error) {
				if slices.Contains(request.Args, "status") {
					return command.Result{
						ExitCode: 2,
						Output:   []byte(`{"initialized":false,"sealed":true,"t":0}`),
					}, errors.New("sealed")
				} else {
					return command.Result{Output: []byte(`{"root_token":"new-token","unseal_keys_b64":["new-key"]}`)}, nil
				}
			}
			path := client.Environment["VAULT_INIT_FILE"]

			// Act
			err := client.Initialize(context.Background())
			info, statErr := os.Stat(path)
			content, readErr := os.ReadFile(path)
			backups, backupErr := readBackups(path)

			// Assert
			Expect(err).To(Succeed())
			Expect(statErr).To(Succeed())
			Expect(info.Mode().Perm()).To(Equal(os.FileMode(0600)))
			Expect(readErr).To(Succeed())
			Expect(string(content)).To(ContainSubstring("new-token"))
			Expect(backupErr).To(Succeed())
			Expect(backups).To(HaveLen(1))
			Expect(backups).To(HaveEach(ContainSubstring("private-token")))
		})

		It("[case] - preserves saved material when Vault is already initialized", func() {
			// Arrange
			path := client.Environment["VAULT_INIT_FILE"]
			original, readErr := os.ReadFile(path)
			Expect(readErr).To(Succeed())
			run = func(_ context.Context, _ command.Request) (command.Result, error) {
				return command.Result{Output: []byte(`{"initialized":true,"sealed":true,"t":2}`)}, nil
			}

			// Act
			err := client.Initialize(context.Background())
			content, readErr := os.ReadFile(path)
			backups, backupErr := readBackups(path)

			// Assert
			Expect(err).To(Succeed())
			Expect(requests).To(HaveLen(1))
			Expect(readErr).To(Succeed())
			Expect(content).To(Equal(original))
			Expect(backupErr).To(Succeed())
			Expect(backups).To(BeEmpty())
		})

		It("[case] - retains received init material when the command fails", func() {
			// Arrange
			var output bytes.Buffer
			client.Output = &output
			run = func(_ context.Context, request command.Request) (command.Result, error) {
				if slices.Contains(request.Args, "status") {
					return command.Result{Output: []byte(`{"initialized":false,"sealed":true}`)}, nil
				} else {
					return command.Result{Output: []byte(`{"root_token":"partial-private-token"`)}, errors.New("connection lost")
				}
			}

			// Act
			err := client.Initialize(context.Background())
			content, readErr := os.ReadFile(client.Environment["VAULT_INIT_FILE"])

			// Assert
			Expect(err).To(MatchError(ContainSubstring("inspect the protected init file")))
			Expect(readErr).To(Succeed())
			Expect(string(content)).To(Equal(`{"root_token":"partial-private-token"`))
			Expect(output.String()).NotTo(ContainSubstring("partial-private-token"))
		})
	})
})
