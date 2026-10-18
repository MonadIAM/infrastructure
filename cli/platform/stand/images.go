package stand

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

var schemaImagePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9./:_-]*@sha256:[a-f0-9]{64}$`)

func (platform Platform) Images(ctx context.Context) error {
	image := platform.Environment["SHARED_SCHEMAS_IMAGE"]
	if !schemaImagePattern.MatchString(image) {
		return errors.New("SHARED_SCHEMAS_IMAGE must be an image reference pinned to a sha256 digest")
	}

	repository, digest, _ := strings.Cut(image, "@")
	if tag := strings.LastIndex(repository, ":"); tag > strings.LastIndex(repository, "/") {
		repository = repository[:tag]
	}

	image = repository + "@" + digest

	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	_, err := platform.Runner.Run(ctx, command.Request{
		Name: "docker",
		Args: []string{"pull", image},
	})
	if err != nil {
		return err
	}

	_, err = platform.Runner.Run(ctx, command.Request{
		Name: "k3d",
		Args: []string{"image", "import", image, "--cluster", platform.Environment["CLUSTER"]},
	})

	return err
}
