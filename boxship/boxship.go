package boxship

import (
	"io"
	"log"

	"github.com/testcontainers/testcontainers-go"
)

//go:generate go run ./update_version.go
func init() {
	testcontainers.Logger = log.New(io.Discard, "", 0)
}
