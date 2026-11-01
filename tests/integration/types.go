//go:build integration

package integration

import (
	"encoding/json"
	"io/fs"
)

type portForwardResult struct {
	Port int
	Err  error
}

type gitopsApplication struct {
	Status struct {
		Conditions []struct {
			Type string `json:"type"`
		} `json:"conditions"`
		Resources []json.RawMessage `json:"resources"`
	} `json:"status"`
}

type resource struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		UID       string `json:"uid"`
	} `json:"metadata"`
	Data map[string][]byte `json:"data"`
	Spec struct {
		NodeName string `json:"nodeName"`
	} `json:"spec"`
	Status struct {
		Conditions []struct {
			Type   string `json:"type"`
			Status string `json:"status"`
		} `json:"conditions"`
	} `json:"status"`
}

type resourceList struct {
	Items []resource `json:"items"`
}

type platformState struct {
	Namespace   string
	Volumes     map[string]string
	Credentials map[string]string
	Material    string
}

type vaultState struct {
	Initialized   bool        `json:"initialized"`
	Sealed        bool        `json:"sealed"`
	FileMode      fs.FileMode `json:"-"`
	DirectoryMode fs.FileMode `json:"-"`
}

var services = []string{"access-control", "identity", "notification", "organization", "hr"}
