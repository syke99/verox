# verox

A chainable `Result`-style type for Go, built on Go 1.27's generic methods.

`verox` wraps the standard `(value, error)` return shape into a `Res[T]` that
you can chain through transforms and fallible steps, with sentinel errors
that stay correctly attributed to the stage that actually produced them —
even across short-circuited chains.

```go
import "github.com/syke99/verox"

func myCallerThatUsesVerox() (Summary, error) {
	return verox.Try(myFirstCaller()).
		Wrap(ErrMyFirstSentinel).
		TryMap(mySecondCaller).
		Wrap(ErrMySecondSentinel).
		Map(toSummary).
		Unwrap()
}
```

## Why

Go's idiomatic `(val, err)` return convention is good, but wrapping errors
with sentinel context for clean handling further up the call stack usually
means either boilerplate at every call site, or double-wrapping/misattributing
errors when a later stage never even ran. `verox` handles that bookkeeping
so a failure in stage one is never mistakenly stamped with stage two's
sentinel.

## Requirements

Go 1.27+ (generic methods). At the time of writing, 1.27 is at release
candidate (`go1.27rc1`+); this module will build against the stable release
once it ships.

## API

- `Try[T](val T, err error) Res[T]` — lift a `(val, err)` pair into a `Res[T]`.
- `(Res[T]) Wrap(sentinel error) Res[T]` — attach a sentinel to the held
  error, unless it was already wrapped by an earlier stage.
- `(Res[T]) Map[U](f func(T) U) Res[U]` — infallible transform.
- `(Res[T]) TryMap[U](f func(T) (U, error)) Res[U]` — fallible step using the
  held value; short-circuits if `r` already failed.
- `(Res[T]) FlatMap[U](f func(T) Res[U]) Res[U]` — chain a step that already
  returns a `Res[U]`, flattening instead of nesting.
- `(Res[T]) Peek(f func(T)) Res[T]` / `PeekErr(f func(error)) Res[T]` —
  observe the value or error without altering the chain.
- `(Res[T]) Is(target error) bool` — `errors.Is` over the held error.
- `(Res[T]) As[E error]() (E, bool)` — `errors.As` over the held error.
- `(Res[T]) Unwrap() (T, error)` — exit the chain back to a plain Go return.

## License

MIT
