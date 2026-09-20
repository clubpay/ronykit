package kit

import (
	"os"

	"github.com/clubpay/ronykit/x/rkit"
)

const envForkChildKey = "RONYKIT_FORK_CHILD"

// we are in the parent process
type child struct {
	pid int
	err error
}

// childID determines if the current process is a child process
func childID() int {
	return rkit.StrToInt(os.Getenv(envForkChildKey))
}
