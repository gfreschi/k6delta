package cli

import (
	"encoding/json"
	"fmt"
	"os"
)

// ExitError signals a non-zero exit code without being a failure.
// main.go inspects this to call os.Exit with the appropriate code.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.Code)
}

// ciOutput writes JSON to stdout.
func ciOutput(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encode CI output: %w", err)
	}
	return nil
}
