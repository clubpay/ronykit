package z

import (
	"os"
	"os/exec"
)

type RunCmdParams struct {
	Dir string
}

func RunCmd(p RunCmdParams, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = p.Dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}
