package terminal

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPaletteLevelColors(t *testing.T) {
	p := palette{enabled: true}

	colored := p.level("INFO")
	assert.True(t, strings.Contains(colored, "\033[32m"))
	assert.True(t, strings.Contains(colored, "INFO"))
}

func TestPaletteDisabled(t *testing.T) {
	p := palette{enabled: false}

	assert.Equal(t, "INFO", p.level("INFO"))
	assert.Equal(t, "dim", p.dim("dim"))
}
