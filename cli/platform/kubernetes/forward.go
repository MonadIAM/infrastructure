package kubernetes

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"
)

func (client Client) Forward(ctx context.Context, namespace, pod, port, kubeconfig string) (string, func(), error) {
	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "--context", client.Context, "-n", namespace,
		"port-forward", "--address=127.0.0.1", "pod/"+pod, ":"+port)
	cmd.WaitDelay = 5 * time.Second
	output, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return "", nil, err
	}

	err = cmd.Start()
	if err != nil {
		cancel()
		return "", nil, err
	}

	closeForward := func() { cancel(); _ = cmd.Wait() }
	ready := make(chan string, 1)
	go func() {
		defer close(ready)
		scanner := bufio.NewScanner(output)
		for scanner.Scan() {
			var localPort int
			count, err := fmt.Sscanf(scanner.Text(), "Forwarding from 127.0.0.1:%d ->", &localPort)
			if err == nil && count == 1 {
				ready <- fmt.Sprintf("127.0.0.1:%d", localPort)
				_, _ = io.Copy(io.Discard, output)
				return
			}
		}
	}()

	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	select {
	case address := <-ready:
		if address != "" {
			return address, closeForward, nil
		}

		closeForward()
		return "", nil, errors.New("kubectl port-forward exited before becoming ready")
	case <-ctx.Done():
		closeForward()
		return "", nil, ctx.Err()
	case <-timer.C:
		closeForward()
		return "", nil, errors.New("kubectl port-forward readiness timeout")
	}
}
