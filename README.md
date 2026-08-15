# Verox
[![Go Reference](https://pkg.go.dev/badge/github.com/syke99/verox.svg)](https://pkg.go.dev/github.com/syke99/verox)

[//]: # ([![Codecov]&#40;https://codecov.io/gh/syke99/verox/branch/main/graph/badge.svg&#41;]&#40;https://codecov.io/gh/syke99/verox&#41;)
[![LICENSE](https://img.shields.io/github/license/syke99/verox)](https://github.com/syke99/verox/blob/main/LICENSE)

Verox, the chainable Result type for Go that keeps sentinel-wrapped errors correctly attributed to the stage that actually produced them

What problem does Verox solve?
=====
Go's `(value, error)` return convention is good, but chaining several fallible steps together and wrapping each one's error with a sentinel for clean handling further up the call stack usually means either a lot of repeated `if err != nil` boilerplate, or bugs that are easy to miss: a later step's sentinel getting stamped onto an earlier step's error that a short-circuited chain never even reached, or a later step's side effects (a DB write, an API call) firing anyway because the short-circuit only happened after the call was already made. Verox's `Res[T]` handles both of these for you: `TryMap`, `Map`, and `FlatMap` never invoke a later step once an earlier one has failed, and `Wrap` won't attribute a sentinel to an error that isn't actually the current stage's own.

</br>

Verox is built on Go 1.27's generic methods, so combinators like `Map`, `TryMap`, and `FlatMap` live directly on `Res[T]` instead of as free functions.

How do I use Verox?
====

### Installing
To install Verox in a repo, simply run

```bash
$ go get github.com/syke99/verox
```

Then you can import the package in any go file you'd like

```go
import "github.com/syke99/verox"
```

### Basic usage

Start a chain with `Try`, passing in the `(value, error)` pair returned by any ordinary Go function:

```go
res := verox.Try(fetchUser(id))
```

Attach a sentinel error with `.Wrap()` so callers further up the stack can check for it with `errors.Is`, regardless of how the underlying error was phrased:

```go
res := verox.Try(fetchUser(id)).
	Wrap(ErrUserLookupFailed)
```

Chain additional fallible steps with `.TryMap()`. Its signature is
`TryMap[U any](f func(val T) (U, error)) Res[U]` — `f` takes the value
currently held by the chain (`T`) and returns the same `(value, error)`
shape any ordinary Go function already returns, just with a possibly
different result type (`U`). Because of that, `f` doesn't need to be a
closure — any existing function whose signature matches `func(T) (U, error)`
can be passed by name directly, the same way `fetchUser` was passed straight
into `Try` above:

```go
// validateUser matches the shape TryMap requires: func(T) (U, error).
// Here T and U both happen to be User, but they don't have to be — TryMap
// can change the type along with validating it, the same way fetchUser
// could just as easily have returned a different type than what comes out
// the other end of validateUser.
func validateUser(u User) (User, error) {
	if u.Email == "" {
		return User{}, errors.New("user has no email")
	}
	return u, nil
}

res := verox.Try(fetchUser(id)).
	Wrap(ErrUserLookupFailed).
	TryMap(validateUser).
	Wrap(ErrUserInvalid)
```

If `fetchUser` already failed, `validateUser` is never called — `TryMap`
short-circuits, so no step past a failure ever runs.

Use `.Map()` for a step that can't fail. Its signature is
`Map[U any](f func(T) U) Res[U]` — no error in the return, just a plain
value transform, so `f` can again be passed by name as long as it matches:

```go
// toProfile matches the shape Map requires: func(T) U — a value in, a
// value out, nothing that can fail.
func toProfile(u User) Profile {
	return Profile{
		DisplayName: u.Name,
		Avatar:      u.AvatarURL,
	}
}

res := verox.Try(fetchUser(id)).
	Wrap(ErrUserLookupFailed).
	TryMap(validateUser).
	Wrap(ErrUserInvalid).
	Map(toProfile)
```

Use `.FlatMap()` for a step that's already built out of its own verox chain
and so already returns a `Res[U]`, rather than a plain `(value, error)`
pair. Its signature is `FlatMap[U any](f func(T) Res[U]) Res[U]`. Reaching
for `.Map()` here instead would leave you with a `Res[Res[U]]` — a `Res`
nested inside a `Res`, needing an extra `.Unwrap()` just to get back to
the value you actually want. `.FlatMap()` flattens that away:

```go
// loadSubscription matches the shape FlatMap requires: func(T) Res[U]. It
// already returns its own Res[Subscription], built from its own internal
// Try/Wrap chain, instead of a plain (value, error) pair.
func loadSubscription(u User) verox.Res[Subscription] {
	return verox.Try(billingClient.GetSubscription(u.ID)).
		Wrap(ErrSubscriptionLookupFailed)
}

res := verox.Try(fetchUser(id)).
	Wrap(ErrUserLookupFailed).
	FlatMap(loadSubscription)
```

Tap into the chain with `.Peek()`/`.PeekErr()` for logging or metrics without disturbing it, and check for specific errors mid-chain with `.Is()`/`.As()`:

```go
res = res.
	Peek(func(p Profile) { log.Printf("resolved profile: %+v", p) }).
	PeekErr(func(err error) { log.Printf("chain failed: %v", err) })

if res.Is(ErrUserInvalid) {
	// handle invalid users specially
}
```

Finally, call `.Unwrap()` to exit back to a plain Go `(value, error)` return:

```go
func GetProfile(id string) (Profile, error) {
	return verox.Try(fetchUser(id)).
		Wrap(ErrUserLookupFailed).
		TryMap(validateUser).
		Wrap(ErrUserInvalid).
		Map(toProfile).
		Unwrap()
}
```

**!!NOTE!!**

Verox requires Go 1.27+ for generic methods. At the time of writing, 1.27 is
still at release candidate (`go1.27rc1`+); until it's stable, building
against this module means pointing `GOTOOLCHAIN` at a matching rc build.

Who?
====

This library was developed by Quinn Millican ([@syke99](https://github.com/syke99))

## License

This repo is under the MIT license, see [LICENSE](LICENSE) for details.
