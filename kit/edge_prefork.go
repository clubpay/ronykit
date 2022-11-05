package kit

import (
	"os"

	"github.com/clubpay/ronykit/kit/utils"
)

const envForkChildKey = "RONYKIT_FORK_CHILD"

// we are in the parent process
type child struct {
	pid int
	err error
}

// childID determines if the current process is a child process
func childID() int {
	return utils.StrToInt(os.Getenv(envForkChildKey))
}
