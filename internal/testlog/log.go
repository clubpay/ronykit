package testlog

import "fmt"

type Log struct{}

func (l Log) Debug(args ...interface{}) {
	fmt.Println(args...) //nolint:forbidigo
}

func (l Log) Debugf(format string, args ...interface{}) {
	fmt.Println(fmt.Sprintf(format, args...)) //nolint:forbidigo
}

func (l Log) Error(args ...interface{}) {
	fmt.Println(args...) //nolint:forbidigo
}

func (l Log) Errorf(format string, args ...interface{}) {
	fmt.Println(fmt.Sprintf(format, args...)) //nolint:forbidigo
}
