package verox

import (
	"errors"
	"fmt"
	"testing"
)

var (
	errBoom     = errors.New("boom")
	errSentinel = errors.New("sentinel")
	errOther    = errors.New("other")
)

func TestTry(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := Try(42, error(nil))
		val, err := r.Unwrap()
		if val != 42 || err != nil {
			t.Fatalf("got val=%v err=%v, want val=42 err=nil", val, err)
		}
	})

	t.Run("failure", func(t *testing.T) {
		r := Try(0, errBoom)
		val, err := r.Unwrap()
		if val != 0 || err != errBoom {
			t.Fatalf("got val=%v err=%v, want val=0 err=%v", val, err, errBoom)
		}
	})
}

func TestWrapErr(t *testing.T) {
	t.Run("wraps an unwrapped error", func(t *testing.T) {
		r := Try(0, errBoom).WrapErr(errSentinel)
		_, err := r.Unwrap()
		if !errors.Is(err, errSentinel) || !errors.Is(err, errBoom) {
			t.Fatalf("expected err to wrap both sentinel and original, got %v", err)
		}
	})

	t.Run("is a no-op on success", func(t *testing.T) {
		r := Try(9, error(nil)).WrapErr(errSentinel)
		val, err := r.Unwrap()
		if err != nil || val != 9 {
			t.Fatalf("got val=%v err=%v, want val=9 err=nil", val, err)
		}
	})

	t.Run("does not double-wrap an already-attributed error", func(t *testing.T) {
		// Simulates the short-circuit path: an error that crossed a stage
		// boundary (wrapped=true) must not pick up a second sentinel.
		stale := Res[int]{err: errBoom, wrapped: true}
		r := stale.WrapErr(errSentinel)
		_, err := r.Unwrap()
		if errors.Is(err, errSentinel) {
			t.Fatalf("expected sentinel NOT to be applied to an already-wrapped error, got %v", err)
		}
		if !errors.Is(err, errBoom) {
			t.Fatalf("expected original error to survive untouched, got %v", err)
		}
	})
}

func TestOr(t *testing.T) {
	t.Run("does not call f on success", func(t *testing.T) {
		called := false
		r := Try(9, error(nil)).Or(func() (int, error) {
			called = true
			return 0, errBoom
		})
		if called {
			t.Fatal("f should not have been called after a prior success")
		}
		val, err := r.Unwrap()
		if val != 9 || err != nil {
			t.Fatalf("got val=%v err=%v, want val=9 err=nil", val, err)
		}
	})

	t.Run("falls back to f on failure", func(t *testing.T) {
		called := false
		r := Try(0, errBoom).Or(func() (int, error) {
			called = true
			return 9, nil
		})
		if !called {
			t.Fatal("expected f to be called after a prior failure")
		}
		val, err := r.Unwrap()
		if val != 9 || err != nil {
			t.Fatalf("got val=%v err=%v, want val=9 err=nil", val, err)
		}
	})

	t.Run("propagates f's own failure", func(t *testing.T) {
		r := Try(0, errBoom).Or(func() (int, error) {
			return 0, errOther
		})
		_, err := r.Unwrap()
		if !errors.Is(err, errOther) {
			t.Fatalf("got err=%v, want %v", err, errOther)
		}
	})

	t.Run("composes correctly with WrapErr after the fallback", func(t *testing.T) {
		// f's failure is fresh, not stale from the original r, so a
		// subsequent WrapErr must still apply to it.
		r := Try(0, errBoom).
			Or(func() (int, error) { return 0, errOther }).
			WrapErr(errSentinel)
		_, err := r.Unwrap()
		if !errors.Is(err, errSentinel) || !errors.Is(err, errOther) {
			t.Fatalf("expected sentinel to wrap f's failure, got %v", err)
		}
		if errors.Is(err, errBoom) {
			t.Fatalf("expected the original (replaced) error NOT to appear, got %v", err)
		}
	})
}

func TestMap(t *testing.T) {
	t.Run("transforms a successful value", func(t *testing.T) {
		r := Try(3, error(nil)).Map(func(n int) string { return fmt.Sprintf("n=%d", n) })
		val, err := r.Unwrap()
		if err != nil || val != "n=3" {
			t.Fatalf("got val=%q err=%v, want val=%q err=nil", val, err, "n=3")
		}
	})

	t.Run("short-circuits without calling f", func(t *testing.T) {
		called := false
		r := Try(0, errBoom).Map(func(n int) string {
			called = true
			return "unreachable"
		})
		if called {
			t.Fatal("f should not have been called after a prior failure")
		}
		_, err := r.Unwrap()
		if !errors.Is(err, errBoom) {
			t.Fatalf("expected original error to propagate, got %v", err)
		}
	})
}

func TestRes_Try(t *testing.T) {
	t.Run("runs f on success", func(t *testing.T) {
		r := Try(4, error(nil)).Try(func(n int) (string, error) {
			return fmt.Sprintf("n=%d", n), nil
		})
		val, err := r.Unwrap()
		if err != nil || val != "n=4" {
			t.Fatalf("got val=%q err=%v, want val=%q err=nil", val, err, "n=4")
		}
	})

	t.Run("propagates an error returned by f", func(t *testing.T) {
		r := Try(4, error(nil)).Try(func(n int) (string, error) {
			return "", errBoom
		})
		_, err := r.Unwrap()
		if !errors.Is(err, errBoom) {
			t.Fatalf("got err=%v, want %v", err, errBoom)
		}
	})

	t.Run("short-circuits without calling f", func(t *testing.T) {
		called := false
		r := Try(0, errBoom).Try(func(n int) (string, error) {
			called = true
			return "unreachable", nil
		})
		if called {
			t.Fatal("f should not have been called after a prior failure")
		}
		_, err := r.Unwrap()
		if !errors.Is(err, errBoom) {
			t.Fatalf("expected original error to propagate, got %v", err)
		}
	})
}

func TestFlatMap(t *testing.T) {
	t.Run("flattens instead of nesting", func(t *testing.T) {
		r := Try(5, error(nil)).FlatMap(func(n int) Res[string] {
			return Try(fmt.Sprintf("n=%d", n), error(nil))
		})
		val, err := r.Unwrap()
		if err != nil || val != "n=5" {
			t.Fatalf("got val=%q err=%v, want val=%q err=nil", val, err, "n=5")
		}
	})

	t.Run("propagates a failure returned by f", func(t *testing.T) {
		r := Try(5, error(nil)).FlatMap(func(n int) Res[string] {
			return Try("", errBoom)
		})
		_, err := r.Unwrap()
		if !errors.Is(err, errBoom) {
			t.Fatalf("got err=%v, want %v", err, errBoom)
		}
	})

	t.Run("short-circuits without calling f", func(t *testing.T) {
		called := false
		r := Try(0, errBoom).FlatMap(func(n int) Res[string] {
			called = true
			return Try("unreachable", error(nil))
		})
		if called {
			t.Fatal("f should not have been called after a prior failure")
		}
		_, err := r.Unwrap()
		if !errors.Is(err, errBoom) {
			t.Fatalf("expected original error to propagate, got %v", err)
		}
	})
}

func TestPeek(t *testing.T) {
	t.Run("fires on success", func(t *testing.T) {
		var seen int
		r := Try(7, error(nil)).Peek(func(n int) { seen = n })
		if seen != 7 {
			t.Fatalf("got seen=%d, want 7", seen)
		}
		val, err := r.Unwrap()
		if val != 7 || err != nil {
			t.Fatalf("Peek must not alter the chain: got val=%v err=%v", val, err)
		}
	})

	t.Run("does not fire on failure", func(t *testing.T) {
		called := false
		r := Try(0, errBoom).Peek(func(n int) { called = true })
		if called {
			t.Fatal("Peek should not fire when r already holds an error")
		}
		_, err := r.Unwrap()
		if !errors.Is(err, errBoom) {
			t.Fatalf("got err=%v, want %v", err, errBoom)
		}
	})
}

func TestPeekErr(t *testing.T) {
	t.Run("fires on failure", func(t *testing.T) {
		var seen error
		r := Try(0, errBoom).PeekErr(func(e error) { seen = e })
		if !errors.Is(seen, errBoom) {
			t.Fatalf("got seen=%v, want %v", seen, errBoom)
		}
		_, err := r.Unwrap()
		if !errors.Is(err, errBoom) {
			t.Fatalf("PeekErr must not alter the chain: got err=%v", err)
		}
	})

	t.Run("does not fire on success", func(t *testing.T) {
		called := false
		r := Try(3, error(nil)).PeekErr(func(e error) { called = true })
		if called {
			t.Fatal("PeekErr should not fire on success")
		}
		val, err := r.Unwrap()
		if val != 3 || err != nil {
			t.Fatalf("got val=%v err=%v, want val=3 err=nil", val, err)
		}
	})
}

func TestIs(t *testing.T) {
	r := Try(0, errBoom).WrapErr(errSentinel)

	t.Run("matches a wrapped sentinel", func(t *testing.T) {
		if !r.Is(errSentinel) {
			t.Fatal("expected Is(errSentinel) to be true")
		}
	})

	t.Run("matches the original error", func(t *testing.T) {
		if !r.Is(errBoom) {
			t.Fatal("expected Is(errBoom) to be true")
		}
	})

	t.Run("does not match an unrelated error", func(t *testing.T) {
		if r.Is(errOther) {
			t.Fatal("expected Is(errOther) to be false")
		}
	})
}

type pathError struct{ path string }

func (e *pathError) Error() string { return "path error: " + e.path }

type temporary interface {
	error
	Temporary() bool
}

type netErr struct{}

func (netErr) Error() string   { return "net broke" }
func (netErr) Temporary() bool { return true }

func TestAs(t *testing.T) {
	t.Run("extracts a concrete pointer error type", func(t *testing.T) {
		r := Try(0, fmt.Errorf("wrap: %w", &pathError{path: "/tmp/x"}))
		pe, ok := r.As[*pathError]()
		if !ok || pe.path != "/tmp/x" {
			t.Fatalf("got ok=%v pe=%v, want ok=true pe.path=/tmp/x", ok, pe)
		}
	})

	t.Run("extracts via an interface-typed target", func(t *testing.T) {
		r := Try(0, fmt.Errorf("wrap: %w", netErr{}))
		te, ok := r.As[temporary]()
		if !ok || !te.Temporary() {
			t.Fatalf("got ok=%v, want ok=true with Temporary()==true", ok)
		}
	})

	t.Run("returns the zero value and false when nothing matches", func(t *testing.T) {
		r := Try(0, errBoom)
		pe, ok := r.As[*pathError]()
		if ok || pe != nil {
			t.Fatalf("got ok=%v pe=%v, want ok=false pe=nil", ok, pe)
		}
	})
}

func TestUnwrap(t *testing.T) {
	t.Run("returns the value on success", func(t *testing.T) {
		val, err := Try("hi", error(nil)).Unwrap()
		if val != "hi" || err != nil {
			t.Fatalf("got val=%q err=%v, want val=hi err=nil", val, err)
		}
	})

	t.Run("returns the error on failure", func(t *testing.T) {
		val, err := Try("", errBoom).Unwrap()
		if val != "" || !errors.Is(err, errBoom) {
			t.Fatalf("got val=%q err=%v, want val=\"\" err=%v", val, err, errBoom)
		}
	})
}

func TestUnwrapOr(t *testing.T) {
	t.Run("returns the value on success", func(t *testing.T) {
		got := Try(9, error(nil)).UnwrapOr(0)
		if got != 9 {
			t.Fatalf("got %v, want 9", got)
		}
	})

	t.Run("returns the default on failure", func(t *testing.T) {
		got := Try(0, errBoom).UnwrapOr(9)
		if got != 9 {
			t.Fatalf("got %v, want 9", got)
		}
	})
}

func TestFold(t *testing.T) {
	t.Run("calls onSuccess, not onFailure", func(t *testing.T) {
		failureCalled := false
		got := Try(9, error(nil)).Fold(
			func(n int) string { return fmt.Sprintf("ok:%d", n) },
			func(err error) string { failureCalled = true; return "fail" },
		)
		if failureCalled {
			t.Fatal("onFailure should not have been called on success")
		}
		if got != "ok:9" {
			t.Fatalf("got %q, want %q", got, "ok:9")
		}
	})

	t.Run("calls onFailure, not onSuccess", func(t *testing.T) {
		successCalled := false
		got := Try(0, errBoom).Fold(
			func(n int) string { successCalled = true; return "ok" },
			func(err error) string { return "fail:" + err.Error() },
		)
		if successCalled {
			t.Fatal("onSuccess should not have been called on failure")
		}
		if got != "fail:boom" {
			t.Fatalf("got %q, want %q", got, "fail:boom")
		}
	})
}

// TestChainAttribution is an end-to-end regression test for the two bugs
// this design went through during development: (1) a later stage's
// function must not execute once an earlier stage has failed, and (2) a
// later stage's sentinel must never be attributed to an earlier stage's
// failure.
func TestChainAttribution(t *testing.T) {
	var errA = errors.New("A")
	var errB = errors.New("B")
	var errC = errors.New("C")

	stageA := func(fail bool) func() (int, error) {
		return func() (int, error) {
			if fail {
				return 0, errors.New("a-broke")
			}
			return 1, nil
		}
	}

	newChain := func(aFails, bFails bool) (called struct{ b, c bool }, result string, err error) {
		bCalled, cCalled := false, false

		val, resErr := Try(stageA(aFails)()).
			WrapErr(errA).
			Try(func(n int) (string, error) {
				bCalled = true
				if bFails {
					return "", errors.New("b-broke")
				}
				return fmt.Sprintf("n=%d", n), nil
			}).
			WrapErr(errB).
			Try(func(s string) (string, error) {
				cCalled = true
				return s + "!", nil
			}).
			WrapErr(errC).
			Unwrap()

		return struct{ b, c bool }{bCalled, cCalled}, val, resErr
	}

	t.Run("stage A fails: B and C never run, only errA attributed", func(t *testing.T) {
		called, _, err := newChain(true, false)
		if called.b || called.c {
			t.Fatalf("expected B and C to be skipped, got called=%+v", called)
		}
		if !errors.Is(err, errA) {
			t.Fatalf("expected errA in chain, got %v", err)
		}
		if errors.Is(err, errB) || errors.Is(err, errC) {
			t.Fatalf("expected errB/errC NOT to be attributed, got %v", err)
		}
	})

	t.Run("stage A ok, B fails: C never runs, only errB attributed", func(t *testing.T) {
		called, _, err := newChain(false, true)
		if !called.b {
			t.Fatal("expected B to have run")
		}
		if called.c {
			t.Fatal("expected C to be skipped")
		}
		if !errors.Is(err, errB) {
			t.Fatalf("expected errB in chain, got %v", err)
		}
		if errors.Is(err, errA) || errors.Is(err, errC) {
			t.Fatalf("expected errA/errC NOT to be attributed, got %v", err)
		}
	})

	t.Run("all succeed", func(t *testing.T) {
		called, val, err := newChain(false, false)
		if !called.b || !called.c {
			t.Fatalf("expected both B and C to run, got called=%+v", called)
		}
		if err != nil {
			t.Fatalf("got err=%v, want nil", err)
		}
		if val != "n=1!" {
			t.Fatalf("got val=%q, want %q", val, "n=1!")
		}
	})
}
