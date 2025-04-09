package boxship

import (
	"context"
	"io"
	"os"
	"regexp"

	"github.com/clubpay/ronykit/boxship/pkg/settings"
	"github.com/testcontainers/testcontainers-go"
)

var regexEnvFileKeys = regexp.MustCompile(`^{{[a-zA-Z0-9_-]{1,32}}}$`)

type Container struct {
	desc *ContainerDesc
	tc   testcontainers.Container

	Name   string
	ID     string
	Logger io.WriteCloser // Container runtime logger
}

func (c *Container) GetContainerID() string {
	return c.tc.GetContainerID()
}

func (c *Container) Run(ctx *Context) error {
	withLogger := ctx.set.GetBool(settings.LogAll)

	ctx.log.Debugf("[%s] container starting", c.Name)
	err := c.tc.Start(ctx.Context())
	if err != nil {
		ctx.log.Warnf("[%s] got error on starting: %v", c.Name, err)

		return err
	}

	if withLogger {
		ctx.Consume(c)
	}

	ctx.log.Debugf("[%s] container started", c.Name)

	if c.desc.AutoExec != nil {
		ctx.log.Debugf("[%s] executing after-start actions", c.Name)
		wd := c.getWorkingDir(c.desc.AutoExec.WorkingDir)
		switch c.desc.AutoExec.RunMode {
		default:
			if err := runAction(wd, c.desc.AutoExec.Exec...); err != nil {
				return err
			}
		case ActionRunModeContainer:
			err = c.execInContainer(ctx.Context(), c.desc.AutoExec.Exec...)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Container) Terminate(ctx *Context) error {
	return c.tc.Terminate(ctx.Context())
}

func (c *Container) getWorkingDir(preset string) string {
	wd := preset
	if wd == "" {
		if c.desc.BuildConfig.Src != nil {
			wd = c.desc.BuildConfig.Src.Context
		} else {
			wd, _ = os.Getwd()
		}
	}

	return wd
}

func (c *Container) execInContainer(ctx context.Context, actions ...[]string) error {
	for _, cmdArgs := range actions {
		_, rd, err := c.tc.Exec(ctx, cmdArgs)
		if err != nil {
			return err
		}
		_, _ = io.Copy(c.Logger, rd)
	}

	return nil
}
