package sync

import (
	"errors"
	"testing"
)

func TestReportAddErr(t *testing.T) {
	t.Run("adds single non-nil error", func(t *testing.T) {
		var r Report
		err := errors.New("failure")
		r.addErr(err)
		if len(r.Errors) != 1 {
			t.Fatalf("expected 1 error, got %d", len(r.Errors))
		}
		if r.Errors[0] != err {
			t.Fatalf("stored error mismatch: got %v", r.Errors[0])
		}
	})

	t.Run("ignores nil error", func(t *testing.T) {
		var r Report
		r.addErr(nil)
		if len(r.Errors) != 0 {
			t.Fatalf("expected no errors, got %d", len(r.Errors))
		}
	})

	t.Run("adds multiple errors preserving order", func(t *testing.T) {
		var r Report
		e1 := errors.New("e1")
		e2 := errors.New("e2")
		e3 := errors.New("e3")
		r.addErr(e1)
		r.addErr(nil)
		r.addErr(e2)
		r.addErr(e3)

		if len(r.Errors) != 3 {
			t.Fatalf("expected 3 errors, got %d", len(r.Errors))
		}
		if r.Errors[0] != e1 || r.Errors[1] != e2 || r.Errors[2] != e3 {
			t.Fatalf("order mismatch: %+v", r.Errors)
		}
	})
}
