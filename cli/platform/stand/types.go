package stand

import (
	"context"
)

type phaseDefinition struct {
	Name string
	Run  func(context.Context) error
}

type operatorDefinition struct {
	Namespace  string
	Deployment string
	Restart    bool
	CRDs       []string
}

type secretCondition struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

type staticSecret struct {
	Kind     string `json:"kind"`
	Metadata struct {
		Name       string `json:"name"`
		Namespace  string `json:"namespace"`
		Generation int64  `json:"generation"`
	} `json:"metadata"`
	Spec struct {
		Destination struct {
			Name string `json:"name"`
		} `json:"destination"`
	} `json:"spec"`
	Status struct {
		LastGeneration int64             `json:"lastGeneration"`
		SecretMAC      string            `json:"secretMAC"`
		Conditions     []secretCondition `json:"conditions"`
	} `json:"status"`
}
