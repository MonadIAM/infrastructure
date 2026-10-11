package debezium

type recordedRequest struct {
	method      string
	path        string
	contentType string
	body        map[string]string
	decodeErr   error
}
