//go:build integration

package integration

import (
	"context"
	"os"
	"os/exec"

	"github.com/joho/godotenv"
)

func (fixture composeFixture) compose(ctx context.Context, args ...string) (string, error) {
	file := fixture.file
	if file == "" {
		file = "testdata/compose.yml"
	}

	options := []string{
		"compose",
		"--project-name", fixture.project,
		"--env-file", "../../../.env.example",
		"-f", file,
	}

	environment, err := godotenv.Read("../../../.env.example")
	if err != nil {
		return "", err
	}

	command := exec.CommandContext(ctx, "docker", append(options, args...)...)
	command.Env = os.Environ()

	for key, value := range environment {
		command.Env = append(command.Env, key+"="+value)
	}

	output, err := command.CombinedOutput()
	return string(output), err
}

func (fixture composeFixture) cli(ctx context.Context, command ...string) (string, error) {
	args := append([]string{"run", "--rm", "--no-deps", "cli"}, command...)

	return fixture.compose(ctx, append(args, "--env-file=")...)
}
