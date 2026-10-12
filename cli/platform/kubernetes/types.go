package kubernetes

type resourceState struct {
	Status map[string]any `json:"status"`
}
