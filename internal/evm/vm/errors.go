package vm

import "errors"

// ErrWriteProtection is raised when a read-only call frame attempts to mutate state.
var ErrWriteProtection = errors.New("write protection")
