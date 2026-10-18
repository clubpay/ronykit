package fixmod

import (
	"os"
	"strconv"
)

func open() *os.File {
	f, err := os.Open("x")
	if err != nil {
		panic(err)
	}

	return f
}

func parse() int {
	n, err := strconv.Atoi("1")
	if err != nil {
		panic(err)
	}

	return n * 2
}

func discard() {
	_, err := strconv.Atoi("1")
	if err != nil {
		panic(err)
	}
}

func remove() {
	err := os.Remove("x")
	if err != nil {
		panic(err)
	}

	if err := os.Remove("y"); err != nil {
		panic(err)
	}
}

func commentInCall() int {
	n, err := strconv.Atoi(
		"1", // literal on purpose
	)
	if err != nil {
		panic(err)
	}

	return n
}

func commented() int {
	n, err := strconv.Atoi("1")
	if err != nil {
		// keep this note
		panic(err)
	}

	return n
}
