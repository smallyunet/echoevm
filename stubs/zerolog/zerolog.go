package zerolog

import "fmt"

// Minimal stub of zerolog for offline builds.

// Level defines log level.
type Level int8

const (
	TraceLevel Level = iota
	DebugLevel
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
	PanicLevel
)

type Event struct {
	out interface{ Write([]byte) (int, error) }
}

func (e *Event) Str(key, val string) *Event             { return e }
func (e *Event) Int(key string, val int) *Event         { return e }
func (e *Event) Any(key string, val interface{}) *Event { return e }
func (e *Event) Msg(msg string) {
	if e.out != nil {
		e.out.Write([]byte(msg + "\n"))
	}
}
func (e *Event) Msgf(format string, args ...interface{}) {
	if e.out != nil {
		fmt.Fprintf(e.out, format+"\n", args...)
	}
}

type Logger struct {
	out interface{ Write([]byte) (int, error) }
}

func New(w interface{}) Logger {
	if wr, ok := w.(interface{ Write([]byte) (int, error) }); ok {
		return Logger{out: wr}
	}
	return Logger{}
}
func Nop() Logger                             { return Logger{} }
func (l Logger) WithLevel(level Level) *Event { return &Event{out: l.out} }
func (l Logger) With() Logger                 { return l }
func (l Logger) Timestamp() Logger            { return l }
func (l Logger) Logger() Logger               { return l }
func (l Logger) Trace() *Event                { return &Event{out: l.out} }
func (l Logger) Debug() *Event                { return &Event{out: l.out} }
func (l Logger) Info() *Event                 { return &Event{out: l.out} }
func (l Logger) Warn() *Event                 { return &Event{out: l.out} }
func (l Logger) Error() *Event                { return &Event{out: l.out} }
func SetGlobalLevel(level Level)              {}

type ConsoleWriter struct {
	Out        interface{}
	TimeFormat string
}
