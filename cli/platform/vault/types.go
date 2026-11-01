package vault

import (
	"errors"
	"time"

	"github.com/MonadIAM/infrastructure/cli/config"
)

type Settings struct {
	Address     string             `koanf:"VAULT_ADDR"`
	InitDir     string             `koanf:"VAULT_INIT_DIR"`
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

type secretDefinition struct {
	Path   string
	Fields []secretField
}

type secretField struct {
	Name        string
	Environment string
	Required    bool
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

	if settings.InitDir == "" || settings.LocalToken == "" || settings.Timeout <= 0 {
		return errors.New("vault paths, token and positive timeout are required")
	} else {
		return nil
	}
}
