package validation

import (
	"testing"

	"github.com/s4f4y4t/go-microservice/pkg/apperror"
)

type signupRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=64"`
}

func TestValidateSuccess(t *testing.T) {
	req := signupRequest{Email: "user@example.com", Password: "longenough"}
	if err := Validate(req); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestValidateRequired(t *testing.T) {
	req := signupRequest{Email: "", Password: "longenough"}
	err := Validate(req)
	if err == nil {
		t.Fatal("Validate() = nil, want error")
	}
	field := findField(err.Fields, "email")
	if field == nil {
		t.Fatalf("expected field error for email, got %+v", err.Fields)
	}
	if field.Message != "Email is required" {
		t.Errorf("Message = %q, want %q", field.Message, "Email is required")
	}
}

func TestValidateEmailFormat(t *testing.T) {
	req := signupRequest{Email: "not-an-email", Password: "longenough"}
	err := Validate(req)
	if err == nil {
		t.Fatal("Validate() = nil, want error")
	}
	field := findField(err.Fields, "email")
	if field == nil || field.Message != "Email must be a valid email address" {
		t.Errorf("field = %+v, want valid email message", field)
	}
}

func TestValidateMinLength(t *testing.T) {
	req := signupRequest{Email: "user@example.com", Password: "short"}
	err := Validate(req)
	if err == nil {
		t.Fatal("Validate() = nil, want error")
	}
	field := findField(err.Fields, "password")
	if field == nil || field.Message != "Password must be at least 8 characters long" {
		t.Errorf("field = %+v, want min length message", field)
	}
}

func TestValidateUsesJSONFieldNames(t *testing.T) {
	req := signupRequest{Password: "longenough"}
	err := Validate(req)
	if err == nil {
		t.Fatal("Validate() = nil, want error")
	}
	if findField(err.Fields, "email") == nil {
		t.Errorf("expected field name to be JSON tag 'email', got %+v", err.Fields)
	}
}

func TestValidateErrorCode(t *testing.T) {
	req := signupRequest{}
	err := Validate(req)
	if err.Code != "INVALID_INPUT" {
		t.Errorf("Code = %v, want INVALID_INPUT", err.Code)
	}
}

func findField(fields []apperror.FieldError, name string) *apperror.FieldError {
	for i := range fields {
		if fields[i].Field == name {
			return &fields[i]
		}
	}
	return nil
}
