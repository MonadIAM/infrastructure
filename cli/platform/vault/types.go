package vault

import (
	"errors"
	"time"

	"github.com/MonadIAM/infrastructure/cli/config"
)

type Settings struct {
	Address     string             `koanf:"VAULT_ADDR"`
	InitDir     string             `koanf:"VAULT_INIT_DIR"`
	PoliciesDir string             `koanf:"VAULT_POLICIES_DIR"`
	LocalToken  string             `koanf:"VAULT_TOKEN"`
	Timeout     time.Duration      `koanf:"INFRA_TIMEOUT"`
	Environment config.Environment `koanf:"-"`
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

func (settings Settings) Validate() error {
	if settings.Address == "" {
		return errors.New("vault address is required")
	}

	if settings.InitDir == "" || settings.PoliciesDir == "" || settings.LocalToken == "" || settings.Timeout <= 0 {
		return errors.New("vault paths, token and positive timeout are required")
	} else {
		return nil
	}
}
