package verox_test

import (
	"errors"
	"fmt"

	"github.com/syke99/verox"
)

var (
	ErrUserLookupFailed = errors.New("user lookup failed")
	ErrUserInvalid      = errors.New("user invalid")
)

type User struct {
	Name  string
	Email string
}

func fetchUser(id string) (User, error) {
	if id == "" {
		return User{}, errors.New("empty id")
	}
	return User{Name: "Ada Lovelace", Email: "ada@example.com"}, nil
}

func validateUser(u User) (User, error) {
	if u.Email == "" {
		return User{}, errors.New("user has no email")
	}
	return u, nil
}

type notFoundError struct{ ID string }

func (e *notFoundError) Error() string { return "user not found: " + e.ID }

// Example shows a full chain from an initial fallible call through to a
// plain Go (value, error) return.
func Example() {
	getName := func(id string) (string, error) {
		return verox.Try(fetchUser(id)).
			WrapErr(ErrUserLookupFailed).
			Try(validateUser).
			WrapErr(ErrUserInvalid).
			Map(func(u User) string { return u.Name }).
			Unwrap()
	}

	name, err := getName("42")
	fmt.Println(name, err)
	// Output:
	// Ada Lovelace <nil>
}

func ExampleTry() {
	res := verox.Try(fetchUser("42"))
	val, err := res.Unwrap()
	fmt.Println(val.Name, err)
	// Output:
	// Ada Lovelace <nil>
}

func ExampleRes_WrapErr() {
	res := verox.Try(fetchUser("")).
		WrapErr(ErrUserLookupFailed)

	_, err := res.Unwrap()
	fmt.Println(errors.Is(err, ErrUserLookupFailed))
	// Output:
	// true
}

func ExampleRes_Map() {
	res := verox.Try(fetchUser("42")).
		Map(func(u User) string { return u.Name })

	name, _ := res.Unwrap()
	fmt.Println(name)
	// Output:
	// Ada Lovelace
}

func ExampleRes_Try() {
	res := verox.Try(fetchUser("42")).
		Try(validateUser)

	val, err := res.Unwrap()
	fmt.Println(val.Email, err)
	// Output:
	// ada@example.com <nil>
}

func ExampleRes_FlatMap() {
	loadGreeting := func(u User) verox.Res[string] {
		return verox.Try(fmt.Sprintf("Hello, %s!", u.Name), error(nil))
	}

	res := verox.Try(fetchUser("42")).
		FlatMap(loadGreeting)

	greeting, _ := res.Unwrap()
	fmt.Println(greeting)
	// Output:
	// Hello, Ada Lovelace!
}

func ExampleRes_Peek() {
	verox.Try(fetchUser("42")).
		Peek(func(u User) { fmt.Println("fetched:", u.Name) })
	// Output:
	// fetched: Ada Lovelace
}

func ExampleRes_PeekErr() {
	verox.Try(fetchUser("")).
		PeekErr(func(err error) { fmt.Println("failed:", err) })
	// Output:
	// failed: empty id
}

func ExampleRes_Is() {
	res := verox.Try(fetchUser("")).
		WrapErr(ErrUserLookupFailed)

	fmt.Println(res.Is(ErrUserLookupFailed))
	// Output:
	// true
}

func ExampleRes_As() {
	res := verox.Try(User{}, fmt.Errorf("lookup failed: %w", &notFoundError{ID: "42"}))

	target, ok := res.As[*notFoundError]()
	fmt.Println(ok, target.ID)
	// Output:
	// true 42
}

func ExampleRes_Unwrap() {
	val, err := verox.Try(fetchUser("42")).
		Try(validateUser).
		Map(func(u User) string { return u.Name }).
		Unwrap()
	fmt.Println(val, err)
	// Output:
	// Ada Lovelace <nil>
}
