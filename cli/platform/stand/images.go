package stand

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

var imagePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9./:_-]*@sha256:[a-f0-9]{64}$`)

func (platform Platform) Images(ctx context.Context) error {
	keys := []string{"SHARED_SCHEMAS_IMAGE"}
	for _, application := range applications {
		keys = append(keys, application.ImageKey)
	}

	for _, key := range keys {
		err := platform.prepareImage(ctx, key)
		if err != nil {
			return err
		}
	}

	return nil
}

func (platform Platform) prepareImage(ctx context.Context, key string) error {
	image := platform.Environment[key]
	if !imagePattern.MatchString(image) {
		return fmt.Errorf("%s must be an image reference pinned to a sha256 digest", key)
	}

	repository, digest, _ := strings.Cut(image, "@")
	if tag := strings.LastIndex(repository, ":"); tag > strings.LastIndex(repository, "/") {
		repository = repository[:tag]
	}

	image = repository + "@" + digest
	cache := repository + ":cache-" + strings.TrimPrefix(digest, "sha256:")

	timeout, err := platform.Environment.Duration("IMAGE_PREPARE_TIMEOUT", "15m")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, err = platform.Runner.Run(ctx, command.Request{
		Name: "docker",
		Args: []string{"pull", image},
	})
	if err != nil {
		return err
	}

	_, err = platform.Runner.Run(ctx, command.Request{
		Name: "docker",
		Args: []string{"tag", image, cache},
	})
	if err != nil {
		return err
	}

	_, err = platform.Runner.Run(ctx, command.Request{
		Name: "k3d",
		Args: []string{"image", "import", cache, "--cluster", platform.Environment["CLUSTER"]},
	})
	if err != nil {
		return err
	}

	return platform.registerImageNodes(ctx, cache, image, digest)
}

func (platform Platform) registerImageNodes(ctx context.Context, cache, image, digest string) error {
	result, err := platform.Runner.Run(ctx, command.Request{
		Name:    "docker",
		Args:    []string{"ps", "--filter", "label=k3d.cluster=" + platform.Environment["CLUSTER"], "--format", `{{.Names}} {{.Label "k3d.role"}}`},
		Capture: true,
	})
	if err != nil {
		return err
	}

	nodes := 0
	for line := range strings.SplitSeq(strings.TrimSpace(string(result.Output)), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && (fields[1] == "server" || fields[1] == "agent") {
			err = platform.registerImage(ctx, fields[0], cache, image, digest)
			if err != nil {
				return err
			}

			nodes++
		}
	}

	if nodes == 0 {
		return fmt.Errorf("stand.Platform.registerImageNodes: no running nodes in cluster %s", platform.Environment["CLUSTER"])
	}

	return nil
}

func (platform Platform) registerImage(ctx context.Context, node, cache, image, digest string) error {
	result, err := platform.Runner.Run(ctx, command.Request{
		Name:    "docker",
		Args:    []string{"exec", node, "ctr", "-n", "k8s.io", "images", "ls", "name==" + cache},
		Capture: true,
	})
	if err != nil {
		return err
	}

	verified := false
	for line := range strings.SplitSeq(string(result.Output), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == cache && fields[2] == digest {
			verified = true
		}
	}

	if !verified {
		return fmt.Errorf("stand.Platform.registerImage: %s on %s does not match %s", cache, node, digest)
	}

	_, err = platform.Runner.Run(ctx, command.Request{
		Name: "docker",
		Args: []string{"exec", node, "ctr", "-n", "k8s.io", "images", "tag", "--force", cache, image},
	})
	if err != nil {
		return err
	}

	_, err = platform.Runner.Run(ctx, command.Request{
		Name:    "docker",
		Args:    []string{"exec", node, "crictl", "inspecti", image},
		Capture: true,
	})

	return err
}
