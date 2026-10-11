package vault

import (
	"context"
	"io"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/command"
	"github.com/MonadIAM/infrastructure/cli/platform/kubernetes"
)

type Client struct {
	Environment config.Environment
	Kubernetes  kubernetes.Client
	Output      io.Writer
}

type Status struct {
	Initialized bool `json:"initialized"`
	Sealed      bool `json:"sealed"`
	Threshold   int  `json:"t"`
}

type material struct {
	RootToken string   `json:"root_token"`
	Keys      []string `json:"unseal_keys_b64"`
}

type credentialDefinition struct {
	Path        string
	UserKey     string
	User        string
	PasswordKey string
	Hash        bool
}

type roleDefinition struct {
	Namespace string
	PolicyKey string
	RoleKey   string
}

type secretResponse struct {
	Data struct {
		Data     map[string]string `json:"data"`
		Metadata struct {
			Version int `json:"version"`
		} `json:"metadata"`
	} `json:"data"`
}

type runnerFunc func(context.Context, command.Request) (command.Result, error)
