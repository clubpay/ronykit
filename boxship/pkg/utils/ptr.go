package utils

func StringPtr(s string) *string {
	x := new(string)
	*x = s

	return x
}
