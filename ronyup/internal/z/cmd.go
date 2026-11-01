package z

import (
	"bytes"
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

	var output bytes.Buffer

	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(output.String())
		if detail != "" {
			return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, detail)
		}

		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}

	return nil
}
