package log

import "github.com/rs/zerolog"

// Logger is a package level logger used for convenience.
var Logger zerolog.Logger = zerolog.Nop()

// SetLogger allows overriding the package level logger.
func SetLogger(l zerolog.Logger) {
	Logger = l
}
