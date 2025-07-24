// CUE schema validation code
package config

import (
	"fmt"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/encoding/yaml"
)

// ValidateWithCue validates a YAML configuration file using a CUE schema file.
func ValidateWithCue(configFile, cueFile string) error {
	ctx := cuecontext.New()

	// Read YAML config
	yamlBytes, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("cannot read YAML config: %w", err)
	}
	configVal := ctx.CompileBytes(yamlBytes, yaml.Parse)

	// Read CUE schema
	schemaBytes, err := os.ReadFile(cueFile)
	if err != nil {
		return fmt.Errorf("cannot read CUE schema: %w", err)
	}
	schemaVal := ctx.CompileBytes(schemaBytes)

	// Merge values with schema
	final := configVal.Unify(schemaVal)
	if final.Err() != nil {
		return fmt.Errorf("schema unify failed: %w", final.Err())
	}

	// Validate final structure
	if err := final.Validate(); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}
	return nil
}