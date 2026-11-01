package stand

import (
	"context"
)

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

type secretCondition struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

type clusterSummary struct {
	Name string `json:"name"`
}

type portBinding struct {
	HostIP   string `json:"HostIp"`
	HostPort string `json:"HostPort"`
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
