// Package errors provides utilities for consistent error handling across the application.
//
// This package offers functions for wrapping errors with context, combining multiple errors,
// and finding the first non-nil error in a sequence.
//
// Example usage:
//
//	// Wrap an error with context
//	if err := operation(); err != nil {
//	    return errors.WrapError(err, "failed to perform operation")
//	}
//
//	// Combine multiple errors
//	errs := []error{err1, err2, err3}
//	if err := errors.CombineErrors(errs); err != nil {
//	    log.Printf("Multiple errors occurred: %v", err)
//	}
//
//	// Get first non-nil error
//	if err := errors.FirstError(err1, err2, err3); err != nil {
//	    return err
//	}
package errors

import (
	"fmt"
)

// WrapError wraps an error with additional context
func WrapError(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s: %w", msg, err)
}

// WrapErrorf wraps an error with formatted context
func WrapErrorf(err error, format string, args ...interface{}) error {
	return WrapError(err, format, args...)
}

// NewError creates a new error with formatted message
func NewError(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

// IsNil checks if an error is nil
func IsNil(err error) bool {
	return err == nil
}

// IsNotNil checks if an error is not nil
func IsNotNil(err error) bool {
	return err != nil
}

// FirstError returns the first non-nil error from a list
func FirstError(errors ...error) error {
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

// CombineErrors combines multiple errors into a single error
func CombineErrors(errors []error) error {
	if len(errors) == 0 {
		return nil
	}

	var nonNilErrors []error
	for _, err := range errors {
		if err != nil {
			nonNilErrors = append(nonNilErrors, err)
		}
	}

	if len(nonNilErrors) == 0 {
		return nil
	}

	if len(nonNilErrors) == 1 {
		return nonNilErrors[0]
	}

	return fmt.Errorf("multiple errors occurred: %v", nonNilErrors)
}
