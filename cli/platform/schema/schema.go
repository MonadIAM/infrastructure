package schema

import "github.com/MonadIAM/infrastructure/cli/config"

func Load(environment config.Environment) (Settings, error) {
	settings := Settings{}

	err := environment.Decode(&settings)
	if err != nil {
		return settings, err
	}

	return settings, settings.Validate()
}
