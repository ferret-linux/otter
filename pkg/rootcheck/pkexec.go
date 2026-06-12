package rootcheck

import "os/exec"

func checkPkexec(name string) error {
	_, err := exec.LookPath(name)
	return err
}
