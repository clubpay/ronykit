package testenv

import "fmt"

type stdLogger struct{}

func (s stdLogger) Debugf(format string, args ...any) {
	fmt.Printf("DEBUG: %s\n", fmt.Sprintf(format, args...))
}

func (s stdLogger) Errorf(format string, args ...any) {
	fmt.Printf("ERROR: %s\n", fmt.Sprintf(format, args...))
}
