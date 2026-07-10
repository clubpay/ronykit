package reflector_test

import (
	"testing"

	"github.com/clubpay/ronykit/kit/utils/reflector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type underscoreTagMessage struct {
	CustomerID int64  `json:"customer_id"`
	Name       string `json:"name,omitempty"`
}

func TestReflectorUnderscoreJSONTag(t *testing.T) {
	m := &underscoreTagMessage{CustomerID: 42, Name: "acme"}
	rObj := reflector.New().Load(m, "json")

	byTag, ok := rObj.ByTag("json")
	require.True(t, ok)

	assert.Equal(t, int64(42), byTag.GetInt64Default(m, "customer_id", 0))
	assert.Equal(t, "acme", byTag.GetStringDefault(m, "name", ""))
}
