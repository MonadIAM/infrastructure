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
