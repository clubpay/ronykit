package rediscluster

import (
	_ "embed"

	"github.com/redis/go-redis/v9"
)

var (
	//go:embed lua/gc.lua
	luaScript string
	luaGC     = redis.NewScript(luaScript)
)
