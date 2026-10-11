package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"time"
)

func (executor Executor) Run(ctx context.Context, request Request) (Result, error) {
	cmd := exec.CommandContext(ctx, request.Name, request.Args...)
	cmd.WaitDelay = 5 * time.Second
	cmd.Stdin = bytes.NewReader(request.Input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if !request.Capture && !request.Sensitive {
		cmd.Stdout = io.MultiWriter(executor.Output, &stdout)
		cmd.Stderr = io.MultiWriter(executor.Errors, &stderr)
	}

	keys := make([]string, 0, len(executor.Environment))
	for key := range executor.Environment {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		cmd.Env = append(cmd.Env, key+"="+executor.Environment[key])
	}

	err := cmd.Run()
	result := Result{
		Output: stdout.Bytes(),
		Stderr: stderr.Bytes(),
	}
	if err == nil {
		return result, nil
	}

	result.ExitCode = -1
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		result.ExitCode = exit.ExitCode()
	}

	contextErr := ctx.Err()
	if contextErr != nil {
		return result, contextErr
	}

	return result, fmt.Errorf("%s failed (exit %d)", request.Name, result.ExitCode)
}

func Pause(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func Poll(ctx context.Context, timeout, interval time.Duration, ready func(context.Context) (bool, error)) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		err := ctx.Err()
		if err != nil {
			return err
		}

		ok, err := ready(ctx)
		if err != nil {
			return err
		}

		if ok {
			return nil
		}

		err = Pause(ctx, interval)
		if err != nil {
			return err
		}
	}
}
