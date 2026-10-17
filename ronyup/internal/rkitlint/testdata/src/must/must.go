package must

import (
	"errors"
	"os"
	"strconv"
)

func open() *os.File {
	f, err := os.Open("x") // want `if err != nil \{ panic\(err\) \} after a call: use rkit.Must`
	if err != nil {
		panic(err)
	}

	return f
}

func parse() int {
	n, err := strconv.Atoi("1") // want `use rkit.Must`
	if err != nil {
		panic(err)
	}

	return n * 2
}

func discard() {
	_, err := strconv.Atoi("1") // want `use rkit.Must`
	if err != nil {
		panic(err)
	}
}

func assert() {
	err := os.Remove("x") // want `use rkit.Assert`
	if err != nil {
		panic(err)
	}
}

func assertInit() {
	if err := os.Remove("x"); err != nil { // want `if err := f\(\); err != nil \{ panic\(err\) \}: use rkit.Assert`
		panic(err)
	}
}

func reversedNil() {
	err := os.Remove("x") // want `use rkit.Assert`
	if nil != err {
		panic(err)
	}
}

func inSwitch(k int) int {
	switch k {
	case 1:
		n, err := strconv.Atoi("1") // want `use rkit.Must`
		if err != nil {
			panic(err)
		}

		return n
	}

	return 0
}

// Not reported below.

func errUsedLater() error {
	n, err := strconv.Atoi("1")
	if err != nil {
		panic(err)
	}

	_ = n

	return err
}

func wrapped() int {
	n, err := strconv.Atoi("1")
	if err != nil {
		panic(errors.New("bad: " + err.Error()))
	}

	return n
}

func hasElse() int {
	n, err := strconv.Atoi("1")
	if err != nil {
		panic(err)
	} else {
		n++
	}

	return n
}

func existingErr() int {
	var err error

	_ = err

	n, err := strconv.Atoi("1")
	if err != nil {
		panic(err)
	}

	return n
}

func notPanic() int {
	n, err := strconv.Atoi("1")
	if err != nil {
		return -1
	}

	return n
}

func shadowedPanic() int {
	panic := func(error) {}

	n, err := strconv.Atoi("1")
	if err != nil {
		panic(err)
	}

	return n
}
