package rediscluster

import (
	"github.com/redis/go-redis/v9"

	_ "embed"
)

var (
	//go:embed lua/gc.lua
	luaScript string
	luaGC     = redis.NewScript(luaScript)
)
