package seaweedfs

import (
	"context"
	"errors"
	"fmt"

	"github.com/minio/minio-go/v7"
)

type RequestError struct {
	Cause error
}

func (failure RequestError) Error() string {
	if errors.Is(failure.Cause, context.Canceled) {
		return context.Canceled.Error()
	}

	if errors.Is(failure.Cause, context.DeadlineExceeded) {
		return context.DeadlineExceeded.Error()
	}

	response := minio.ToErrorResponse(failure.Cause)
	if response.StatusCode != 0 {
		return fmt.Sprintf("S3 request failed with status %d", response.StatusCode)
	} else {
		return "S3 request failed"
	}
}

func (failure RequestError) Unwrap() error {
	return failure.Cause
}
