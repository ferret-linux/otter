package rootcheck

import (
	"fmt"
	"os/exec"
)

func checkRun0(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("run0 not found: %w", err)
	}
	return nil
}
