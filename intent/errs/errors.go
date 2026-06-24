package errs

import (
	"errors"
	"fmt"

	"github.com/clubpay/ronykit/rony/errs"
)

var (
	ErrSessionNotFound      = notFound("session not found")
	ErrModelNotFound        = notFound("model not found")
	ErrToolNotFound         = notFound("tool not found")
	ErrSkillNotFound        = notFound("skill not found")
	ErrRecordNotFound       = notFound("memory record not found")
	ErrKnowledgeNotFound    = notFound("knowledge entry not found")
	ErrEmptyPool            = invalid("llm pool is empty")
	ErrUnsupportedOperation = invalid("operation not supported")
	ErrMaxToolIterations    = invalid("max tool iterations exceeded")
	ErrNotConnected         = invalid("mcp server not connected")
	ErrAgentNotFound        = notFound("agent not found")
	ErrChannelNotFound      = notFound("channel not found")
	ErrTaskNotFound         = notFound("task not found")
	ErrInvalidRequest       = invalid("invalid request")
)

func notFound(msg string) error {
	return errs.B().Code(errs.NotFound).Msg(msg).Err()
}

func invalid(msg string) error {
	return errs.B().Code(errs.InvalidArgument).Msg(msg).Err()
}

func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}

	return errs.Wrap(err, msg)
}

func IsNotFound(err error) bool {
	var e *errs.Error
	if !errors.As(err, &e) {
		return false
	}

	return e.Code == errs.NotFound
}

func SessionNotFound(id string) error {
	return errs.B().Code(errs.NotFound).Msg(fmt.Sprintf("session %q not found", id)).Err()
}

func ModelNotFound(id string) error {
	return errs.B().Code(errs.NotFound).Msg(fmt.Sprintf("model %q not found", id)).Err()
}

func ToolNotFound(name string) error {
	return errs.B().Code(errs.NotFound).Msg(fmt.Sprintf("tool %q not found", name)).Err()
}

func SkillNotFound(name string) error {
	return errs.B().Code(errs.NotFound).Msg(fmt.Sprintf("skill %q not found", name)).Err()
}

func AgentNotFound(id string) error {
	return errs.B().Code(errs.NotFound).Msg(fmt.Sprintf("agent %q not found", id)).Err()
}

func ChannelNotFound(id string) error {
	return errs.B().Code(errs.NotFound).Msg(fmt.Sprintf("channel %q not found", id)).Err()
}

func TaskNotFound(id string) error {
	return errs.B().Code(errs.NotFound).Msg(fmt.Sprintf("task %q not found", id)).Err()
}
