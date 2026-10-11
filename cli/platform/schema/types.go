package schema

import (
	"fmt"
	"slices"
	"strings"

	"github.com/MonadIAM/infrastructure/cli/config"
)

type Settings struct {
	Validity      string
	Compatibility string
	Integrity     string
}

func Load(environment config.Environment) (Settings, error) {
	settings := Settings{
		Validity:      environment["SCHEMA_REGISTRY_GLOBAL_VALIDITY"],
		Compatibility: environment["SCHEMA_REGISTRY_GLOBAL_COMPATIBILITY"],
		Integrity:     environment["SCHEMA_REGISTRY_GLOBAL_INTEGRITY"],
	}
	return settings, settings.Validate()
}

func (settings Settings) Validate() error {
	if !slices.Contains([]string{"NONE", "SYNTAX_ONLY", "FULL"}, settings.Validity) {
		return fmt.Errorf("invalid SCHEMA_REGISTRY_GLOBAL_VALIDITY")
	}

	if !slices.Contains([]string{"NONE", "BACKWARD", "BACKWARD_TRANSITIVE", "FORWARD", "FORWARD_TRANSITIVE", "FULL", "FULL_TRANSITIVE"}, settings.Compatibility) {
		return fmt.Errorf("invalid SCHEMA_REGISTRY_GLOBAL_COMPATIBILITY")
	}

	for _, level := range strings.Split(settings.Integrity, ",") {
		if !slices.Contains([]string{"NONE", "FULL", "NO_DUPLICATES", "REFS_EXIST", "ALL_REFS_MAPPED"}, strings.TrimSpace(level)) {
			return fmt.Errorf("invalid SCHEMA_REGISTRY_GLOBAL_INTEGRITY")
		}
	}

	return nil
}
