package errors

import (
	"errors"
	"testing"
)

func TestWrapError(t *testing.T) {
	baseErr := errors.New("base error")
	wrapped := WrapError(baseErr, "context: %s", "additional info")

	if wrapped == nil {
		t.Error("Expected non-nil error")
	}

	if !errors.Is(wrapped, baseErr) {
		t.Error("Wrapped error should unwrap to base error")
	}
}

func TestWrapErrorNil(t *testing.T) {
	wrapped := WrapError(nil, "context")

	if wrapped != nil {
		t.Error("Wrapping nil should return nil")
	}
}

func TestFirstError(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	// Should return first non-nil error
	result := FirstError(nil, err1, err2)
	if result != err1 {
		t.Error("Expected first non-nil error")
	}

	// Should return nil if all are nil
	result = FirstError(nil, nil, nil)
	if result != nil {
		t.Error("Expected nil when all errors are nil")
	}
}

func TestCombineErrors(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	// Empty list
	result := CombineErrors([]error{})
	if result != nil {
		t.Error("Expected nil for empty list")
	}

	// Single error
	result = CombineErrors([]error{err1})
	if result != err1 {
		t.Error("Expected single error to be returned as-is")
	}

	// Multiple errors
	result = CombineErrors([]error{err1, err2})
	if result == nil {
		t.Error("Expected combined error")
	}

	// With nil errors
	result = CombineErrors([]error{nil, err1, nil, err2})
	if result == nil {
		t.Error("Expected combined error ignoring nils")
	}
}
