package stand

import (
	"context"
	"io"

	"github.com/MonadIAM/infrastructure/cli/config"
	"github.com/MonadIAM/infrastructure/cli/platform/command"
	"github.com/MonadIAM/infrastructure/cli/platform/helm"
	"github.com/MonadIAM/infrastructure/cli/platform/kubernetes"
)

type Platform struct {
	Environment config.Environment
	Runner      command.Runner
	Output      io.Writer
	Kubernetes  kubernetes.Client
	Helm        helm.Client
}

type phaseDefinition struct {
	Name string
	Run  func(context.Context) error
}

type operatorDefinition struct {
	Release    string
	Namespace  string
	Deployment string
	Restart    bool
	CRDs       []string
}

type secretDefinition struct {
	Namespace   string
	Name        string
	Destination string
}

type staticSecret struct {
	Metadata struct {
		Generation int64 `json:"generation"`
	} `json:"metadata"`
	Status struct {
		LastGeneration int64             `json:"lastGeneration"`
		SecretMAC      string            `json:"secretMAC"`
		Conditions     []secretCondition `json:"conditions"`
	} `json:"status"`
}

type secretCondition struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

type runnerFunc func(context.Context, command.Request) (command.Result, error)

type clusterSummary struct {
	Name string `json:"name"`
}
