package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/s4f4y4t/go-microservice/pkg/response"
	"github.com/s4f4y4t/go-microservice/services/auth/internal/clients/user"
)

func newJSONRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) response.ApiResponse {
	t.Helper()
	var body response.ApiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, body = %s", err, w.Body.String())
	}
	return body
}

func newTestHandler(repo UserLookup, store *fakeTokenStore, issuer *fakeIssuer) *Handler {
	svc := newTestService(repo, store, issuer)
	return NewHandler(svc, false)
}

func TestHandlerRegisterSuccess(t *testing.T) {
	repo := &fakeUserRepo{
		createFn: func(ctx context.Context, u *user.User) (*user.User, error) {
			u.ID = 1
			return u, nil
		},
	}
	h := newTestHandler(repo, newFakeTokenStore(), &fakeIssuer{})

	req := newJSONRequest(t, http.MethodPost, "/v1/auth/register", RegisterDTO{
		Name: "Ann", Email: "ann@example.com", Password: "longenough",
	})
	w := httptest.NewRecorder()
	h.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}
	if body := decodeBody(t, w); !body.Success {
		t.Errorf("body = %+v", body)
	}
}

func TestHandlerRegisterValidationError(t *testing.T) {
	h := newTestHandler(&fakeUserRepo{}, newFakeTokenStore(), &fakeIssuer{})

	req := newJSONRequest(t, http.MethodPost, "/v1/auth/register", RegisterDTO{
		Name: "Ann", Email: "not-an-email", Password: "short",
	})
	w := httptest.NewRecorder()
	h.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandlerRegisterConflict(t *testing.T) {
	repo := &fakeUserRepo{
		existsByEmailFn: func(ctx context.Context, email string) (bool, error) { return true, nil },
	}
	h := newTestHandler(repo, newFakeTokenStore(), &fakeIssuer{})

	req := newJSONRequest(t, http.MethodPost, "/v1/auth/register", RegisterDTO{
		Name: "Ann", Email: "ann@example.com", Password: "longenough",
	})
	w := httptest.NewRecorder()
	h.Register(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestHandlerLoginSetsRefreshCookie(t *testing.T) {
	repo := &fakeUserRepo{
		getByEmailFn: func(ctx context.Context, email string) (*user.User, error) {
			return &user.User{ID: 1, Email: email, Password: hashPassword(t, "correct-password")}, nil
		},
	}
	h := newTestHandler(repo, newFakeTokenStore(), &fakeIssuer{})

	req := newJSONRequest(t, http.MethodPost, "/v1/auth/login", LoginDTO{
		Email: "ann@example.com", Password: "correct-password",
	})
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != refreshCookieName || cookies[0].Value == "" {
		t.Fatalf("expected refresh cookie to be set, got %+v", cookies)
	}
	if !cookies[0].HttpOnly {
		t.Errorf("expected refresh cookie to be HttpOnly")
	}
}

func TestHandlerLoginInvalidCredentials(t *testing.T) {
	repo := &fakeUserRepo{
		getByEmailFn: func(ctx context.Context, email string) (*user.User, error) {
			return &user.User{ID: 1, Email: email, Password: hashPassword(t, "correct-password")}, nil
		},
	}
	h := newTestHandler(repo, newFakeTokenStore(), &fakeIssuer{})

	req := newJSONRequest(t, http.MethodPost, "/v1/auth/login", LoginDTO{
		Email: "ann@example.com", Password: "wrong-password",
	})
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}

func TestHandlerRefreshMissingCookie(t *testing.T) {
	h := newTestHandler(&fakeUserRepo{}, newFakeTokenStore(), &fakeIssuer{})

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", nil)
	w := httptest.NewRecorder()
	h.Refresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}

func TestHandlerRefreshRotatesCookie(t *testing.T) {
	store := newFakeTokenStore()
	store.tokens["old-refresh"] = 5
	h := newTestHandler(&fakeUserRepo{}, store, &fakeIssuer{})

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "old-refresh"})
	w := httptest.NewRecorder()
	h.Refresh(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Value == "old-refresh" {
		t.Fatalf("expected rotated refresh cookie, got %+v", cookies)
	}
}

func TestHandlerLogoutClearsCookieRegardlessOfExistingCookie(t *testing.T) {
	h := newTestHandler(&fakeUserRepo{}, newFakeTokenStore(), &fakeIssuer{})

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	w := httptest.NewRecorder()
	h.Logout(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge >= 0 {
		t.Fatalf("expected expiring refresh cookie, got %+v", cookies)
	}
}
