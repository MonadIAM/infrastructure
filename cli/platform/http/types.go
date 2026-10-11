package http

import (
	"fmt"
	"net/http"
)

type Client struct {
	URL      string
	Token    string
	User     string
	Password string
	HTTP     *http.Client
}

type StatusError struct {
	Code int
}

func (failure StatusError) Error() string {
	return fmt.Sprintf("HTTP request failed with status %d", failure.Code)
}
