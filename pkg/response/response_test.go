package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/s4f4y4t/go-microservice/pkg/apperror"
)

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	Success(w, http.StatusOK, "ok", map[string]string{"id": "1"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body ApiResponse
	decode(t, w, &body)
	if !body.Success || body.Message != "ok" || body.StatusCode != http.StatusOK {
		t.Errorf("body = %+v", body)
	}
}

func TestSuccessWithMeta(t *testing.T) {
	w := httptest.NewRecorder()
	SuccessWithMeta(w, http.StatusOK, "listed", []int{1, 2}, map[string]int{"total": 2})

	var body ApiResponse
	decode(t, w, &body)
	if body.Meta == nil {
		t.Errorf("expected Meta to be set, got %+v", body)
	}
}

func TestNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	NoContent(w)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if w.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", w.Body.String())
	}
}

func TestErrorWithAppError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/things/1", nil)

	Error(w, r, apperror.NotFound("thing not found"))

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var body ApiResponse
	decode(t, w, &body)
	if body.Success {
		t.Errorf("expected Success = false, got %+v", body)
	}
	if body.Error == nil || body.Error.Code != apperror.CodeNotFound || body.Error.Message != "thing not found" {
		t.Errorf("Error = %+v", body.Error)
	}
}

func TestErrorWithUnknownErrorHidesDetails(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/things/1", nil)

	Error(w, r, errors.New("db connection dropped"))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var body ApiResponse
	decode(t, w, &body)
	if body.Error == nil || body.Error.Code != apperror.CodeInternal {
		t.Errorf("Error = %+v", body.Error)
	}
	if body.Error.Message == "db connection dropped" {
		t.Errorf("internal error details leaked to client: %q", body.Error.Message)
	}
}

func TestErrorWithValidationFields(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/things", nil)

	fields := []apperror.FieldError{{Field: "email", Message: "is required"}}
	Error(w, r, apperror.Validation("validation failed", fields))

	var body ApiResponse
	decode(t, w, &body)
	if len(body.Error.Fields) != 1 || body.Error.Fields[0].Field != "email" {
		t.Errorf("Fields = %+v", body.Error.Fields)
	}
}

func decode(t *testing.T, w *httptest.ResponseRecorder, into *ApiResponse) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), into); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, body = %s", err, w.Body.String())
	}
}
