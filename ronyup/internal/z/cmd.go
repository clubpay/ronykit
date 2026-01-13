package z

import (
	"context"
	"os"
	"os/exec"
)

type RunCmdParams struct {
	Dir string
}

func RunCmd(ctx context.Context, p RunCmdParams, name string, args ...string) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = p.Dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}
