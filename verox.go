// Package verox provides a chainable Result-style type for Go, with
// sentinel-error wrapping that correctly attributes failures to the stage
// that actually produced them.
package verox

import (
	"errors"
	"fmt"
)

// Res holds either a successful value of type T or an error produced along
// the way.
type Res[T any] struct {
	val     T
	err     error
	wrapped bool
}

// Try lifts a (val, err) pair — the standard Go function-return shape —
// into a Res.
func Try[T any](val T, err error) Res[T] {
	return Res[T]{val: val, err: err}
}

// Wrap attaches sentinel to the held error, unless it has already been
// wrapped by an earlier stage in the chain (see TryMap/FlatMap/Map).
func (r Res[T]) Wrap(sentinel error) Res[T] {
	if r.err != nil && !r.wrapped {
		r.err = fmt.Errorf("%w: %w", sentinel, r.err)
	}
	return r
}

// Map applies an infallible transform to a successful value.
func (r Res[T]) Map[U any](f func(T) U) Res[U] {
	if r.err != nil {
		return Res[U]{err: r.err, wrapped: true}
	}
	return Res[U]{val: f(r.val)}
}

// TryMap applies a fallible step using the held value, short-circuiting if
// r already carries a failure.
func (r Res[T]) TryMap[U any](f func(val T) (U, error)) Res[U] {
	if r.err != nil {
		return Res[U]{err: r.err, wrapped: true}
	}
	val, err := f(r.val)
	return Res[U]{val: val, err: err}
}

// FlatMap chains a step that already returns a Res[U], flattening the
// result instead of nesting it.
func (r Res[T]) FlatMap[U any](f func(T) Res[U]) Res[U] {
	if r.err != nil {
		return Res[U]{err: r.err, wrapped: true}
	}
	return f(r.val)
}

// Peek observes a successful value without altering the chain.
func (r Res[T]) Peek(f func(T)) Res[T] {
	if r.err == nil {
		f(r.val)
	}
	return r
}

// PeekErr observes a failure without altering the chain.
func (r Res[T]) PeekErr(f func(error)) Res[T] {
	if r.err != nil {
		f(r.err)
	}
	return r
}

// Is reports whether the held error matches target, per errors.Is.
func (r Res[T]) Is(target error) bool {
	return errors.Is(r.err, target)
}

// As extracts a concrete error type E from the held error, per errors.As.
func (r Res[T]) As[E error]() (target E, ok bool) {
	ok = errors.As(r.err, &target)
	return
}

// Unwrap returns the held value and error.
func (r Res[T]) Unwrap() (T, error) {
	return r.val, r.err
}
