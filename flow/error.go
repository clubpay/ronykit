package flow

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/trace"
)

type errWrap struct {
	err     error
	traceID string
}

func (e errWrap) Error() string {
	return fmt.Sprintf("traceID[%s]: %v", e.traceID, e.err)
}

func WrapError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	return &errWrap{
		err:     err,
		traceID: trace.SpanFromContext(ctx).SpanContext().TraceID().String(),
	}
}
