//go:build integration

package integration

type taskStatus struct {
	State string `json:"state"`
}

type archiveObject struct {
	size int64
	rows string
}
