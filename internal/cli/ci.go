package cli

import (
	"encoding/json"
	"fmt"
	"os"
)

// ciOutput writes JSON to stdout.
func ciOutput(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encode CI output: %w", err)
	}
	return nil
}
