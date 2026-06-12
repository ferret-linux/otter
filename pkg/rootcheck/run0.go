package rootcheck

import "os/exec"

func checkRun0(name string) error {
	_, err := exec.LookPath(name)
	return err
}
