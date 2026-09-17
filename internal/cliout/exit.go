// Package cliout provides shared CLI exit codes and output helpers (ROADMAP Phase A).
//
// Exit codes:
//
//	0 ExitOK           — OK / ready
//	1 ExitUsage        — usage / flag error
//	2 ExitIncomplete   — workspace incomplete
//	3 ExitInconsistent — CR above threshold
//	4 ExitProposals    — proposals pending (blocked for clean checks)
//	5 ExitIO           — I/O or engine failure
//
// Output: human tables on a TTY; JSON when --json is set or stdout is not a TTY.
// Return *ExitError from Cobra RunE; main maps via Code(err) → os.Exit.
package cliout

import (
	"errors"
	"fmt"
)

// Exit code contract for scripts and CI.
const (
	ExitOK           = 0
	ExitUsage        = 1
	ExitIncomplete   = 2
	ExitInconsistent = 3
	ExitProposals    = 4
	ExitIO           = 5
)

// Aliases without the Exit prefix (same values).
const (
	OK           = ExitOK
	Usage        = ExitUsage
	Incomplete   = ExitIncomplete
	Inconsistent = ExitInconsistent
	Proposals    = ExitProposals
	IO           = ExitIO
)

// ExitError carries a process exit code for main to honor.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit %d", e.Code)
}

func (e *ExitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NewExitError builds an ExitError with a formatted message.
func NewExitError(code int, format string, args ...any) *ExitError {
	return &ExitError{Code: code, Err: fmt.Errorf(format, args...)}
}

// Errorf builds an ExitError with a formatted message (returns error).
func Errorf(code int, format string, args ...any) error {
	return NewExitError(code, format, args...)
}

// Wrap returns an ExitError with the given code, or nil if err is nil.
func Wrap(code int, err error) error {
	if err == nil {
		return nil
	}
	return &ExitError{Code: code, Err: err}
}

// Code returns the exit code for err.
//   - nil → ExitOK
//   - *ExitError (possibly wrapped) → its Code
//   - any other error → ExitUsage (Cobra flag/usage + legacy plain errors)
func Code(err error) int {
	if err == nil {
		return ExitOK
	}
	var ee *ExitError
	if errors.As(err, &ee) && ee != nil {
		return ee.Code
	}
	return ExitUsage
}

// CodeOf is an alias for Code.
func CodeOf(err error) int { return Code(err) }
