package z

import (
	"context"
	"os"
	"os/exec"
)

type RunCmdParams struct {
	Dir string
	ENV map[string]string
}

func RunCmd(ctx context.Context, p RunCmdParams, name string, args ...string) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = p.Dir
	for k, v := range p.ENV {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}
