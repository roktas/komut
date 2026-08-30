package komut

import "fmt"

type ErrorCode string

const (
	ErrInvalidInvocation  ErrorCode = "invalid-invocation"
	ErrInvalidCommand     ErrorCode = "invalid-command"
	ErrCommandNotFound    ErrorCode = "command-not-found"
	ErrUnsafeProjectPath  ErrorCode = "unsafe-project-path"
	ErrInvalidCommandFile ErrorCode = "invalid-command-file"
	ErrUnterminatedQuote  ErrorCode = "unterminated-quote"
	ErrMissingArgument    ErrorCode = "missing-argument"
)

type Error struct {
	Code    ErrorCode
	Command string
	Detail  string
}

func (e *Error) Error() string {
	if e.Command != "" && e.Detail != "" {
		return fmt.Sprintf("%s: %s", e.Command, e.Detail)
	}
	if e.Command != "" {
		return e.Command
	}
	return e.Detail
}

func fail(code ErrorCode, command, detail string) error {
	return &Error{Code: code, Command: command, Detail: detail}
}
