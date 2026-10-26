// Copied from go-engine (https://github.com/akrck02/go-engine/tree/main/failure, MIT licence).
// TODO(akrck02): share it as its own module instead of a copy in each project.

// Package failure gives every error llmcp returns a code: a stable name, as
// "hub.session-not-found", that a program can check and a person can search for.
//
// A code is declared once, as a package-level variable, and is itself an error:
//
//	var SessionNotFound = failure.Define("hub.session-not-found", "no session has that name")
//
// Returning the code alone allocates nothing and comparing it is comparing two pointers, as
// cheap as Go's own sentinel errors. When the error needs its details (which session, what went
// wrong underneath), Wrap makes one error with them, on a request and never in a loop.
//
// Callers check an error by its code, wrapped or not:
//
//	if failure.Is(err, hub.SessionNotFound) { … }
//
// errors.Is works the same way, since an error made by Wrap answers to its code.
package failure

import "errors"

// Code is a kind of error: its stable name and what it means. Declare codes with Define.
type Code struct {
	name    string
	meaning string
}

// Define declares a code. The name is the package's name, a dot and the error in a few words
// joined by dashes; the meaning says what went wrong, for a person.
func Define(name, meaning string) *Code {
	return &Code{name: name, meaning: meaning}
}

// Name is the code's stable name, as "hub.session-not-found".
func (code *Code) Name() string { return code.name }

// Meaning is what the code means, for a person.
func (code *Code) Meaning() string { return code.meaning }

// Error is the code's name and meaning: a code is an error on its own.
func (code *Code) Error() string { return code.name + ": " + code.meaning }

// Error is a coded error with its details: what the code means here, and the error underneath.
type Error struct {
	Code    *Code
	Details string // what is particular to this error: a file, a name
	Cause   error  // the error underneath, if any
}

// Wrap makes an error of a code with its details and the error underneath (nil if none).
func Wrap(code *Code, details string, cause error) error {
	return &Error{Code: code, Details: details, Cause: cause}
}

// Error is the code's name, the details, and the error underneath.
func (failed *Error) Error() string {
	message := failed.Code.name
	if failed.Details != "" {
		message += ": " + failed.Details
	}
	if failed.Cause != nil {
		message += ": " + failed.Cause.Error()
	}
	return message
}

// Unwrap is the error underneath, so errors.Is and errors.As reach it.
func (failed *Error) Unwrap() error { return failed.Cause }

// Is tells whether the error has a code: errors.Is(err, code) asks it.
func (failed *Error) Is(target error) bool {
	code, isCode := target.(*Code)
	return isCode && failed.Code == code
}

// Is tells whether an error, or any error it wraps, has a code.
func Is(err error, code *Code) bool { return errors.Is(err, code) }

// CodeOf is the code of an error, the outermost one that has one; nil for an error without a
// code (one from outside llmcp).
func CodeOf(err error) *Code {
	var code *Code
	if errors.As(err, &code) {
		return code
	}
	var failed *Error
	if errors.As(err, &failed) {
		return failed.Code
	}
	return nil
}
