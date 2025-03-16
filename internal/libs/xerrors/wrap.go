package xerrors

import "fmt"

func Wrap(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}

func Errorf(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}
