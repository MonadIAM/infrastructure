package schema

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSchema(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Schema Registry settings")
}

var _ = Describe("[Config] - Schema Registry", func() {
	Describe("[Method] - Settings.Validate", func() {
		Context("[Behavior] - combined integrity rules", func() {
			It("[case] - accepts comma-separated rules with surrounding whitespace", func() {
				// Arrange
				settings := Settings{
					Validity:      "FULL",
					Compatibility: "BACKWARD",
					Integrity:     "REFS_EXIST, NO_DUPLICATES",
				}

				// Act
				err := settings.Validate()

				// Assert
				Expect(err).To(Succeed())
			})
		})

		DescribeTable("[Behavior] - rejects unknown rules in each setting",
			func(settings Settings, key string) {
				// Act
				err := settings.Validate()

				// Assert
				Expect(err).To(MatchError("invalid " + key))
			},
			Entry("[case] - validity", Settings{Validity: "unknown", Compatibility: "BACKWARD", Integrity: "FULL"}, "SCHEMA_REGISTRY_GLOBAL_VALIDITY"),
			Entry("[case] - compatibility", Settings{Validity: "FULL", Compatibility: "unknown", Integrity: "FULL"}, "SCHEMA_REGISTRY_GLOBAL_COMPATIBILITY"),
			Entry("[case] - second integrity rule", Settings{Validity: "FULL", Compatibility: "BACKWARD", Integrity: "REFS_EXIST, unknown"}, "SCHEMA_REGISTRY_GLOBAL_INTEGRITY"),
		)
	})
})
