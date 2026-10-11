package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func ValidateURL(address string) error {
	parsed, err := url.Parse(address)
	if err != nil || parsed.Hostname() == "" {
		return errors.New("endpoint must be an HTTP(S) URL")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("endpoint must use HTTP or HTTPS")
	}

	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("endpoint must not contain credentials, query or fragment")
	} else {
		return nil
	}
}

func New(address string) Client {
	return Client{
		URL: strings.TrimRight(address, "/"),
		HTTP: &http.Client{
			Timeout:       30 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

func (client Client) Do(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, method, client.URL+path, bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("cannot create HTTP request")
	}

	request.Header.Set("Content-Type", "application/json")
	if client.Token != "" {
		request.Header.Set("X-Vault-Token", client.Token)
	}

	if client.User != "" {
		request.SetBasicAuth(client.User, client.Password)
	}

	response, err := client.HTTP.Do(request)
	if err != nil {
		contextErr := ctx.Err()
		if contextErr == nil {
			return nil, errors.New("HTTP endpoint is unreachable")
		} else {
			return nil, contextErr
		}
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, StatusError{Code: response.StatusCode}
	}

	data, err := io.ReadAll(io.LimitReader(response.Body, 16<<20))
	if err == nil {
		return data, nil
	} else {
		return nil, errors.New("cannot read HTTP response")
	}
}

func (client Client) JSON(ctx context.Context, method, path string, input, output any) error {
	var body []byte
	var err error
	if input != nil {
		body, err = json.Marshal(input)
		if err != nil {
			return errors.New("cannot encode HTTP request")
		}
	}

	data, err := client.Do(ctx, method, path, body)
	if err != nil {
		return err
	}

	if output != nil && len(data) != 0 {
		err := json.Unmarshal(data, output)
		if err != nil {
			return errors.New("invalid JSON response")
		}
	}

	return nil
}

func IsStatus(err error, code int) bool {
	var status StatusError

	return errors.As(err, &status) && status.Code == code
}

func Wait(ctx context.Context, interval time.Duration, ready func() error) error {
	for {
		err := ctx.Err()
		if err != nil {
			return err
		}

		err = ready()
		if err == nil {
			return nil
		}

		if IsStatus(err, 401) || IsStatus(err, 403) {
			return err
		}

		timer := time.NewTimer(interval)

		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
