package ratelimit

import (
	_ "embed"

	"github.com/redis/go-redis/v9"
)

var (
	//go:embed lua/allow_n.lua
	luaAllowNScript string
	//go:embed lua/allow_atmost_n.lua
	luaAllowAtMostScript string
)

var (
	luaAllowN      *redis.Script
	luaAllowAtMost *redis.Script
)

func init() {
	luaAllowN = redis.NewScript(luaAllowNScript)
	luaAllowAtMost = redis.NewScript(luaAllowAtMostScript)
}
