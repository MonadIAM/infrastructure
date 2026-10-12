package config

import (
	"errors"
	"io/fs"

	"github.com/go-viper/mapstructure/v2"
)

type sourceError struct {
	cause error
}

func (failure sourceError) Error() string {
	var decodeError *mapstructure.DecodeError
	if errors.As(failure.cause, &decodeError) {
		return "invalid configuration field " + decodeError.Name()
	}

	var pathError *fs.PathError
	if errors.As(failure.cause, &pathError) {
		return pathError.Err.Error()
	} else {
		return "cannot parse configuration"
	}
}

func (failure sourceError) Unwrap() error {
	return failure.cause
}
