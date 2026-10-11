package vault

import (
	"errors"
	"time"

	"github.com/MonadIAM/infrastructure/cli/config"
	infrahttp "github.com/MonadIAM/infrastructure/cli/platform/http"
)

type Settings struct {
	Address     string
	InitDir     string
	PoliciesDir string
	LocalToken  string
	Timeout     time.Duration
	Environment config.Environment
}

type sealStatus struct {
	Initialized bool `json:"initialized"`
	Sealed      bool `json:"sealed"`
}

type initResult struct {
	Keys      []string `json:"keys"`
	RootToken string   `json:"root_token"`
}

type mount struct {
	Type    string            `json:"type"`
	Options map[string]string `json:"options"`
}

type mountList struct {
	Data map[string]mount `json:"data"`
}

type mountDefinition struct {
	Endpoint string
	Name     string
	Type     string
	Options  map[string]string
}

type roleDefinition struct {
	Name            string   `json:"-"`
	Policies        []string `json:"token_policies"`
	TokenTTL        string   `json:"token_ttl"`
	TokenMaxTTL     string   `json:"token_max_ttl"`
	SecretIDTTL     string   `json:"secret_id_ttl"`
	SecretIDNumUses int      `json:"secret_id_num_uses"`
}

type secretDefinition struct {
	Path   string
	Fields []secretField
}

type secretField struct {
	Name        string
	Environment string
	Required    bool
}

type keyDefinition struct {
	KeySize int
	Name    string
	Type    string
	Signing bool
}

type keyResponse struct {
	Data struct {
		Type string `json:"type"`
	} `json:"data"`
}

type tokenResponse struct {
	Data struct {
		Policies []string `json:"policies"`
		TTL      int      `json:"ttl"`
	} `json:"data"`
}

func Load(environment config.Environment) (Settings, error) {
	settings := Settings{
		Address:     environment.Get("VAULT_ADDR", "http://vault:8200"),
		InitDir:     environment.Get("VAULT_INIT_DIR", "/vault/init"),
		PoliciesDir: environment.Get("VAULT_POLICIES_DIR", "vault/policies"),
		LocalToken:  environment["VAULT_TOKEN"],
		Environment: environment,
	}
	var err error
	settings.Timeout, err = environment.Duration("INFRA_TIMEOUT", "2m")
	if err == nil {
		return settings, settings.Validate()
	} else {
		return settings, err
	}
}

func (settings Settings) Validate() error {
	err := infrahttp.ValidateURL(settings.Address)
	if err != nil {
		return err
	}

	if settings.InitDir == "" || settings.PoliciesDir == "" || settings.LocalToken == "" || settings.Timeout <= 0 {
		return errors.New("vault paths, token and positive timeout are required")
	} else {
		return nil
	}
}
