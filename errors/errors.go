package errors

import "fmt"

type UniversalError struct {
	ErrorText string
	Module    string
}

func (f UniversalError) Error() string {
	return fmt.Sprintf("%s %s", f.ErrorText, f.Module)
}
