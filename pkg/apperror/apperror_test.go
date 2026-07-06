package apperror

import (
	"errors"
	"net/http"
	"testing"

	"gorm.io/gorm"
)

func TestConstructors(t *testing.T) {
	tests := []struct {
		name    string
		err     *AppError
		code    Code
		message string
	}{
		{"NotFound", NotFound("missing"), CodeNotFound, "missing"},
		{"InvalidInput", InvalidInput("bad"), CodeInvalidInput, "bad"},
		{"Conflict", Conflict("dup"), CodeConflict, "dup"},
		{"Unauthorized", Unauthorized("nope"), CodeUnauthorized, "nope"},
		{"Forbidden", Forbidden("denied"), CodeForbidden, "denied"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("Code = %v, want %v", tt.err.Code, tt.code)
			}
			if tt.err.Message != tt.message {
				t.Errorf("Message = %q, want %q", tt.err.Message, tt.message)
			}
		})
	}
}

func TestInternal(t *testing.T) {
	cause := errors.New("boom")
	err := Internal(cause)

	if err.Code != CodeInternal {
		t.Errorf("Code = %v, want %v", err.Code, CodeInternal)
	}
	if !errors.Is(err, cause) {
		t.Errorf("expected Internal error to unwrap to cause")
	}
	if got, want := err.Error(), "internal server error: boom"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestErrorWithoutCause(t *testing.T) {
	err := NotFound("resource not found")
	if got, want := err.Error(), "resource not found"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestValidation(t *testing.T) {
	fields := []FieldError{{Field: "email", Message: "invalid"}}
	err := Validation("validation failed", fields)

	if err.Code != CodeInvalidInput {
		t.Errorf("Code = %v, want %v", err.Code, CodeInvalidInput)
	}
	if len(err.Fields) != 1 || err.Fields[0].Field != "email" {
		t.Errorf("Fields = %+v, want [{email invalid}]", err.Fields)
	}
}

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		code Code
		want int
	}{
		{CodeNotFound, http.StatusNotFound},
		{CodeInvalidInput, http.StatusBadRequest},
		{CodeConflict, http.StatusConflict},
		{CodeUnauthorized, http.StatusUnauthorized},
		{CodeForbidden, http.StatusForbidden},
		{CodeTooManyRequests, http.StatusTooManyRequests},
		{CodeInternal, http.StatusInternalServerError},
		{Code("UNKNOWN"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			e := &AppError{Code: tt.code}
			if got := e.HTTPStatus(); got != tt.want {
				t.Errorf("HTTPStatus() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFrom(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		if From(nil) != nil {
			t.Errorf("From(nil) should be nil")
		}
	})

	t.Run("already AppError passes through", func(t *testing.T) {
		original := Conflict("dup")
		if got := From(original); got != original {
			t.Errorf("From() = %v, want same instance %v", got, original)
		}
	})

	t.Run("wrapped AppError unwraps via errors.As", func(t *testing.T) {
		original := Forbidden("denied")
		wrapped := errors.Join(errors.New("context"), original)
		got := From(wrapped)
		if got != original {
			t.Errorf("From() = %v, want %v", got, original)
		}
	})

	t.Run("gorm not found maps to NotFound", func(t *testing.T) {
		got := From(gorm.ErrRecordNotFound)
		if got.Code != CodeNotFound {
			t.Errorf("Code = %v, want %v", got.Code, CodeNotFound)
		}
	})

	t.Run("unknown error maps to Internal", func(t *testing.T) {
		cause := errors.New("weird")
		got := From(cause)
		if got.Code != CodeInternal {
			t.Errorf("Code = %v, want %v", got.Code, CodeInternal)
		}
		if !errors.Is(got, cause) {
			t.Errorf("expected Internal error to preserve cause")
		}
	})
}

func TestUnwrap(t *testing.T) {
	cause := errors.New("cause")
	err := Internal(cause)
	if err.Unwrap() != cause {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), cause)
	}
}
