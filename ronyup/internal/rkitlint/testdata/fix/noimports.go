package fixmod

func get() (int, error) { return 1, nil }

func useGet() int {
	v, err := get()
	if err != nil {
		panic(err)
	}

	return v + 1
}
