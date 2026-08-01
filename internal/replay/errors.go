package replay

import "errors"

type ErrorKind string

const (
	ErrorInvalid     ErrorKind = "invalid"
	ErrorNotFound    ErrorKind = "not_found"
	ErrorConflict    ErrorKind = "conflict"
	ErrorUpstream    ErrorKind = "upstream"
	ErrorUnavailable ErrorKind = "unavailable"
)

type ReplayError struct {
	Kind ErrorKind
	Err  error
}

func (e *ReplayError) Error() string { return e.Err.Error() }
func (e *ReplayError) Unwrap() error { return e.Err }

func NewError(kind ErrorKind, err error) error {
	if err == nil {
		return nil
	}
	return &ReplayError{Kind: kind, Err: err}
}

func ErrorKindOf(err error) ErrorKind {
	var target *ReplayError
	if errors.As(err, &target) {
		return target.Kind
	}
	return ErrorInvalid
}
