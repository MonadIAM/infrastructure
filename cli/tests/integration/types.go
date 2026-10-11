//go:build integration

package integration

import "io/fs"

type clusterFixture struct {
	name        string
	directory   string
	environment []string
}

type resource struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		UID       string `json:"uid"`
	} `json:"metadata"`
	Data map[string][]byte `json:"data"`
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
