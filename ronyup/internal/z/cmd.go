package z

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type RunCmdParams struct {
	Dir string
	ENV map[string]string
}

func RunCmd(ctx context.Context, p RunCmdParams, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = p.Dir

	if len(p.ENV) > 0 {
		cmd.Env = os.Environ()
		for k, v := range p.ENV {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}

	return nil
}
