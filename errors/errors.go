package errors

type Code string

type Place string

type Error struct {
	Code  Code
	Place Place
	Cause error
}

func (e *Error) Error() string {
	return string(e.Code)
}

func New(code Code) *Error {
	return &Error{
		Code: code,
	}
}

func NewWithPlace(code Code, place Place) *Error {
	return &Error{
		Code:  code,
		Place: place,
	}
}

func (e *Error) Unwrap() error {
	return e.Cause
}
