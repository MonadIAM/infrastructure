package helm

type Release struct {
	Name      string
	Chart     string
	Version   string
	Namespace string
	Values    string
	Arguments []string
}
