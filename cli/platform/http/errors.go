package http

import (
	"context"
	"errors"
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

	return "HTTP request failed"
}

func (failure RequestError) Unwrap() error {
	return failure.Cause
}
