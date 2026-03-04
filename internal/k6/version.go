package k6runner

import "os/exec"

// SupportsJSONStreaming checks if the installed k6 supports --out json.
// Returns false if k6 is not installed or not accessible.
func SupportsJSONStreaming() (bool, error) {
	cmd := exec.Command("k6", "version")
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	// k6 has supported JSON output since very early versions.
	// This is mainly to detect if k6 is even installed.
	return len(out) > 0, nil
}
