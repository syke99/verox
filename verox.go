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
	val T
	err error
	// wrapped marks an error as already attributed to the stage that
	// produced it, so a later WrapErr call doesn't stamp it with a sentinel
	// that belongs to a stage that never ran.
	wrapped bool
}

// Try lifts a (val, err) pair — the standard Go function-return shape —
// into a Res.
func Try[T any](val T, err error) Res[T] {
	return Res[T]{val: val, err: err}
}

// WrapErr attaches sentinel to the held error via %w, so errors.Is and
// errors.As still see both sentinel and the original error. It is a no-op
// on success, and a no-op if the error has already been wrapped by an
// earlier stage in the chain (see Try/FlatMap/Map).
func (r Res[T]) WrapErr(sentinel error) Res[T] {
	if r.err != nil && !r.wrapped {
		r.err = fmt.Errorf("%w: %w", sentinel, r.err)
	}
	return r
}

// Or falls back to an alternate computation if r already holds an error,
// such as trying a different data source. If r holds a successful value,
// f is never called and r is returned unchanged.
func (r Res[T]) Or(f func() Res[T]) Res[T] {
	if r.err == nil {
		return r
	}
	return f()
}

// Map applies an infallible transform to a successful value. If r already
// holds an error, f is never called and the error propagates unchanged.
func (r Res[T]) Map[U any](f func(T) U) Res[U] {
	if r.err != nil {
		return Res[U]{err: r.err, wrapped: true}
	}
	return Res[U]{val: f(r.val)}
}

// Try applies a fallible step using the held value. If r already holds an
// error, f is never called and the error propagates unchanged; this is
// what lets a chain stop running further steps as soon as one fails.
//
// Try shares its name with the package-level Try function; they don't
// collide since one is reached through a receiver (r.Try(...)) and the
// other through the package (verox.Try(...)).
func (r Res[T]) Try[U any](f func(val T) (U, error)) Res[U] {
	if r.err != nil {
		return Res[U]{err: r.err, wrapped: true}
	}
	val, err := f(r.val)
	return Res[U]{val: val, err: err}
}

// FlatMap chains a step that already returns a Res[U], flattening the
// result instead of nesting it. If r already holds an error, f is never
// called and the error propagates unchanged.
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

// UnwrapOr returns the held value, or def if r holds an error.
func (r Res[T]) UnwrapOr(def T) T {
	if r.err != nil {
		return def
	}
	return r.val
}

// Fold resolves r into a single value by calling exactly one of onSuccess
// or onFailure, depending on whether r holds a value or an error.
func (r Res[T]) Fold[U any](onSuccess func(T) U, onFailure func(error) U) U {
	if r.err != nil {
		return onFailure(r.err)
	}
	return onSuccess(r.val)
}
