package exitcode

import (
	"errors"
	"fmt"
	"testing"
)

func TestExitCodeFromTypedErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil error", nil, Success},
		{"general error", General("fail", nil), GeneralError},
		{"usage error", Usage("bad flag"), UsageError},
		{"auth error", Auth("bad key", nil), AuthFailure},
		{"not found error", NotFoundError("issue not found"), NotFound},
		{"plain error", errors.New("something"), GeneralError},
		{"wrapped not found", fmt.Errorf("resolving: %w", NotFoundError("issue not found")), NotFound},
		{"wrapped auth", fmt.Errorf("outer: %w", Auth("bad key", nil)), AuthFailure},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExitCode(tt.err)
			if got != tt.want {
				t.Errorf("ExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestErrorMessages(t *testing.T) {
	t.Run("with wrapped error", func(t *testing.T) {
		inner := errors.New("connection refused")
		err := General("API request failed", inner)
		if err.Error() != "API request failed: connection refused" {
			t.Errorf("Error() = %q, want %q", err.Error(), "API request failed: connection refused")
		}
		if !errors.Is(err, inner) {
			t.Error("expected Unwrap to return inner error")
		}
	})

	t.Run("without wrapped error", func(t *testing.T) {
		err := Usage("missing required flag --pipeline")
		if err.Error() != "missing required flag --pipeline" {
			t.Errorf("Error() = %q, want %q", err.Error(), "missing required flag --pipeline")
		}
	})

	t.Run("formatted error", func(t *testing.T) {
		err := Generalf("rate limited — retry after %d seconds", 30)
		if err.Error() != "rate limited — retry after 30 seconds" {
			t.Errorf("Error() = %q, want %q", err.Error(), "rate limited — retry after 30 seconds")
		}
	})
}

func TestExitCodeConstants(t *testing.T) {
	if Success != 0 {
		t.Errorf("Success = %d, want 0", Success)
	}
	if GeneralError != 1 {
		t.Errorf("GeneralError = %d, want 1", GeneralError)
	}
	if UsageError != 2 {
		t.Errorf("UsageError = %d, want 2", UsageError)
	}
	if AuthFailure != 3 {
		t.Errorf("AuthFailure = %d, want 3", AuthFailure)
	}
	if NotFound != 4 {
		t.Errorf("NotFound = %d, want 4", NotFound)
	}
}
