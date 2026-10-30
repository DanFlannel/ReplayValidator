// Package errors provides a consistent interface for using errors.
// It also supports slog structured logging attributes; i.e. structured errors.
// It is also a drop-in replacement for the standard library errors package.
package errors

import (
	pkgerrors "github.com/pkg/errors"
)

// Sentinel returns a package-level error value without logging it.
//
// New logs at construction, which is right for an error that has just happened
// and wrong for one being *declared*. A `var ErrX = errors.New(...)` at package
// scope runs at init, so every service that imports the package prints a phantom
// ERROR line at startup — on Cloud Run with scale-to-zero, on every cold start,
// for errors that have not occurred. It buries real failures and would page on
// nothing once error reporting is wired.
//
// Use this for sentinels compared with errors.Is; use New at the point a failure
// actually occurs.
func Sentinel(msg string) error {
	return structured{
		err: pkgerrors.New(msg),
	}
}

// New creates a structured error carrying key-value attrs. It does NOT log,
// and it does NOT printf.
func New(msg string, attrs ...any) error {
	return structured{
		err:   pkgerrors.New(msg),
		attrs: attrs,
	}
}

// Wrap returns a new error wrapping the provided with additional
// structured fields.
//
//nolint:wrapcheck,inamedparam // This function does custom wrapping and errors.
func Wrap(err error, msg string, attrs ...any) error {
	if err == nil {
		panic("wrap nil error")
	}

	// Support error types that do their own wrapping.
	if wrapper, ok := err.(interface{ Wrap(string, ...any) error }); ok {
		return wrapper.Wrap(msg, attrs...)
	}

	var inner structured
	if As(err, &inner) {
		attrs = append(attrs, inner.attrs...) // Append inner attributes
	}

	return structured{
		err:   pkgerrors.Wrap(err, msg),
		attrs: attrs,
	}
}
