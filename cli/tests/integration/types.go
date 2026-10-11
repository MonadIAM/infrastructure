//go:build integration

package integration

type composeFixture struct {
	project string
	file    string
}

type taskStatus struct {
	State string `json:"state"`
}

type connectorStatus struct {
	Connector taskStatus   `json:"connector"`
	Tasks     []taskStatus `json:"tasks"`
}

type archiveObject struct {
	size int64
	rows string
}

func (status connectorStatus) running() bool {
	return status.Connector.State == "RUNNING" &&
		len(status.Tasks) == 1 &&
		status.Tasks[0].State == "RUNNING"
}
