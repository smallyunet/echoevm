package vm

import "errors"

// ErrWriteProtection is raised when a read-only call frame attempts to mutate state.
var ErrWriteProtection = errors.New("write protection")

// ErrReturnDataOutOfBounds is raised when RETURNDATACOPY reads outside the
// return-data buffer. EIP-211 defines this as an exceptional halt rather than
// a zero-padded copy.
var ErrReturnDataOutOfBounds = errors.New("return data out of bounds")
