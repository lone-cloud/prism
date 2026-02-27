package delivery

import "errors"

type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string {
	return e.Err.Error()
}

func (e *PermanentError) Unwrap() error {
	return e.Err
}

func NewPermanentError(err error) error {
	return &PermanentError{Err: err}
}

func IsPermanent(err error) bool {
	var permErr *PermanentError
	return errors.As(err, &permErr)
}
