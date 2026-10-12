package schema

import (
	"errors"
	"slices"
	"strings"
)

type Settings struct {
	Validity      string `koanf:"SCHEMA_REGISTRY_GLOBAL_VALIDITY"`
	Compatibility string `koanf:"SCHEMA_REGISTRY_GLOBAL_COMPATIBILITY"`
	Integrity     string `koanf:"SCHEMA_REGISTRY_GLOBAL_INTEGRITY"`
}

func (settings Settings) Validate() error {
	if !slices.Contains([]string{"NONE", "SYNTAX_ONLY", "FULL"}, settings.Validity) {
		return errors.New("invalid SCHEMA_REGISTRY_GLOBAL_VALIDITY")
	}

	if !slices.Contains([]string{"NONE", "BACKWARD", "BACKWARD_TRANSITIVE", "FORWARD", "FORWARD_TRANSITIVE", "FULL", "FULL_TRANSITIVE"}, settings.Compatibility) {
		return errors.New("invalid SCHEMA_REGISTRY_GLOBAL_COMPATIBILITY")
	}

	for _, level := range strings.Split(settings.Integrity, ",") {
		if !slices.Contains([]string{"NONE", "FULL", "NO_DUPLICATES", "REFS_EXIST", "ALL_REFS_MAPPED"}, strings.TrimSpace(level)) {
			return errors.New("invalid SCHEMA_REGISTRY_GLOBAL_INTEGRITY")
		}
	}

	return nil
}
