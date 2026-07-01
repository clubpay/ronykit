package logkit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewExporterTerminal(t *testing.T) {
	exp, err := NewExporter("test-service", WithTerminal(true))
	require.NoError(t, err)
	require.NotNil(t, exp)
}
