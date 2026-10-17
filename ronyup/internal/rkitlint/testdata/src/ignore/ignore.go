package ignore

import "strconv"

func sameLine() int {
	n, err := strconv.Atoi("1") //rkitcheck:ignore
	if err != nil {
		panic(err)
	}

	return n
}

func lineAbove() int {
	//rkitcheck:ignore startup only
	n, err := strconv.Atoi("1")
	if err != nil {
		panic(err)
	}

	return n
}

//rkitcheck:ignore
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}

	return v
}

func nolintIsNotOurs() int {
	n, err := strconv.Atoi("1") //nolint:rkitcheck // want `use rkit.Must`
	if err != nil {
		panic(err)
	}

	return n
}
