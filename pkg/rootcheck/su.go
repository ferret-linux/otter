package rootcheck

import "os/exec"

func checkSu(name string) error {
	_, err := exec.LookPath(name)
	return err
}
