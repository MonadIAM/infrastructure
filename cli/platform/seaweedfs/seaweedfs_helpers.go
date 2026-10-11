package seaweedfs

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func testSettings() Settings {
	return Settings{
		Endpoint:       "http://seaweedfs:8333",
		Region:         "us-east-1",
		AdminKey:       "admin",
		AdminSecret:    "admin-secret",
		ArchiverKey:    "archiver",
		ArchiverSecret: "archiver-secret",
		ConfigDir:      "../../../platform/seaweedfs/config",
		Timeout:        time.Second,
		Buckets: []Bucket{
			{Name: "audit-archive", Days: 1095},
			{Name: "change-log-archive", Days: 365},
		},
	}
}

func (fixture *s3Fixture) serve(response http.ResponseWriter, request *http.Request) {
	fixture.mutex.Lock()
	defer fixture.mutex.Unlock()

	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(response, "cannot read request", http.StatusBadRequest)
		return
	}

	fixture.requests = append(fixture.requests, s3Request{
		Method: request.Method,
		Path:   request.URL.Path,
		Query:  request.URL.RawQuery,
		Body:   string(body),
		Lock:   request.Header.Get("X-Amz-Bucket-Object-Lock-Enabled"),
	})

	response.Header().Set("Content-Type", "application/xml")
	if fixture.status != 0 {
		response.WriteHeader(fixture.status)
		_, _ = io.WriteString(response, "<Error><Code>AccessDenied</Code><Message>private-server-detail</Message></Error>")
		return
	}

	if request.URL.Path == "/" {
		_, _ = io.WriteString(response, "<ListAllMyBucketsResult><Buckets/></ListAllMyBucketsResult>")
		return
	}

	if request.Method == http.MethodPut && request.URL.RawQuery == "" && fixture.existing {
		response.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(response, "<Error><Code>BucketAlreadyOwnedByYou</Code></Error>")
		return
	}

	if request.Method == http.MethodGet && request.URL.Query().Has("object-lock") {
		days := fixture.lockDays
		if days == 0 {
			days = 1095
			if request.URL.Path == "/change-log-archive/" || request.URL.Path == "/change-log-archive" {
				days = 365
			}
		}

		_, _ = fmt.Fprintf(response, `<ObjectLockConfiguration><ObjectLockEnabled>Enabled</ObjectLockEnabled><Rule><DefaultRetention><Mode>COMPLIANCE</Mode><Days>%d</Days></DefaultRetention></Rule></ObjectLockConfiguration>`, days)
		return
	}

	response.WriteHeader(http.StatusOK)
}

func (fixture *s3Fixture) snapshot() []s3Request {
	fixture.mutex.Lock()
	defer fixture.mutex.Unlock()

	return append([]s3Request(nil), fixture.requests...)
}
