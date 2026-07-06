package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/s4f4y4t/go-microservice/pkg/apperror"
	"github.com/s4f4y4t/go-microservice/services/auth/internal/clients/user"
	"golang.org/x/crypto/bcrypt"
)

// fakeUserRepo is an in-memory stand-in for UserLookup.
type fakeUserRepo struct {
	existsByEmailFn func(ctx context.Context, email string) (bool, error)
	getByEmailFn    func(ctx context.Context, email string) (*user.User, error)
	createFn        func(ctx context.Context, u *user.User) (*user.User, error)
}

func (f *fakeUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if f.existsByEmailFn != nil {
		return f.existsByEmailFn(ctx, email)
	}
	return false, nil
}

func (f *fakeUserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	if f.getByEmailFn != nil {
		return f.getByEmailFn(ctx, email)
	}
	return nil, apperror.NotFound("user not found")
}

func (f *fakeUserRepo) Create(ctx context.Context, u *user.User) (*user.User, error) {
	if f.createFn != nil {
		return f.createFn(ctx, u)
	}
	return u, nil
}

// fakeTokenStore is an in-memory stand-in for token.Store.
type fakeTokenStore struct {
	tokens  map[string]int
	saveErr error
	userErr error
	delErr  error
	deleted []string
}

func newFakeTokenStore() *fakeTokenStore {
	return &fakeTokenStore{tokens: map[string]int{}}
}

func (f *fakeTokenStore) Save(ctx context.Context, tok string, userID int, expiry time.Duration) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.tokens[tok] = userID
	return nil
}

func (f *fakeTokenStore) UserID(ctx context.Context, tok string) (int, error) {
	if f.userErr != nil {
		return 0, f.userErr
	}
	id, ok := f.tokens[tok]
	if !ok {
		return 0, apperror.Unauthorized("invalid or expired refresh token")
	}
	return id, nil
}

func (f *fakeTokenStore) Delete(ctx context.Context, tok string) error {
	if f.delErr != nil {
		return f.delErr
	}
	f.deleted = append(f.deleted, tok)
	delete(f.tokens, tok)
	return nil
}

// fakeIssuer is an in-memory stand-in for token.AccessIssuer.
type fakeIssuer struct {
	issueErr error
}

func (f *fakeIssuer) Issue(userID int, expiry time.Duration) (string, error) {
	if f.issueErr != nil {
		return "", f.issueErr
	}
	return "access-token", nil
}

func hashPassword(t *testing.T, plain string) string {
	t.Helper()
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword() error = %v", err)
	}
	return string(hashed)
}

func newTestService(repo UserLookup, store *fakeTokenStore, issuer *fakeIssuer) *Service {
	return NewService(repo, store, issuer, 15*time.Minute, 24*time.Hour)
}

func TestRegisterSuccess(t *testing.T) {
	var created *user.User
	repo := &fakeUserRepo{
		existsByEmailFn: func(ctx context.Context, email string) (bool, error) { return false, nil },
		createFn: func(ctx context.Context, u *user.User) (*user.User, error) {
			created = u
			u.ID = 1
			return u, nil
		},
	}
	svc := newTestService(repo, newFakeTokenStore(), &fakeIssuer{})

	got, err := svc.Register(context.Background(), RegisterDTO{Name: "Ann", Email: "ann@example.com", Password: "longenough"})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if got.ID != 1 || got.Email != "ann@example.com" {
		t.Errorf("Register() = %+v", got)
	}
	if created.Password == "longenough" {
		t.Errorf("password was not hashed before storage")
	}
	if bcrypt.CompareHashAndPassword([]byte(created.Password), []byte("longenough")) != nil {
		t.Errorf("stored hash does not match original password")
	}
}

func TestRegisterEmailAlreadyExists(t *testing.T) {
	repo := &fakeUserRepo{
		existsByEmailFn: func(ctx context.Context, email string) (bool, error) { return true, nil },
	}
	svc := newTestService(repo, newFakeTokenStore(), &fakeIssuer{})

	_, err := svc.Register(context.Background(), RegisterDTO{Name: "Ann", Email: "ann@example.com", Password: "longenough"})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeConflict {
		t.Fatalf("Register() error = %v, want Conflict", err)
	}
}

func TestRegisterExistsCheckError(t *testing.T) {
	repoErr := errors.New("db down")
	repo := &fakeUserRepo{
		existsByEmailFn: func(ctx context.Context, email string) (bool, error) { return false, repoErr },
	}
	svc := newTestService(repo, newFakeTokenStore(), &fakeIssuer{})

	_, err := svc.Register(context.Background(), RegisterDTO{Name: "Ann", Email: "ann@example.com", Password: "longenough"})
	if !errors.Is(err, repoErr) {
		t.Fatalf("Register() error = %v, want wrapped %v", err, repoErr)
	}
}

func TestLoginSuccess(t *testing.T) {
	repo := &fakeUserRepo{
		getByEmailFn: func(ctx context.Context, email string) (*user.User, error) {
			return &user.User{ID: 7, Email: email, Password: hashPassword(t, "correct-password")}, nil
		},
	}
	store := newFakeTokenStore()
	svc := newTestService(repo, store, &fakeIssuer{})

	pair, err := svc.Login(context.Background(), LoginDTO{Email: "ann@example.com", Password: "correct-password"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if pair.AccessToken != "access-token" {
		t.Errorf("AccessToken = %q, want %q", pair.AccessToken, "access-token")
	}
	if pair.RefreshToken == "" {
		t.Errorf("RefreshToken is empty")
	}
	if store.tokens[pair.RefreshToken] != 7 {
		t.Errorf("refresh token not stored against correct user id: %v", store.tokens)
	}
	if pair.AccessExpiresIn != int((15 * time.Minute).Seconds()) {
		t.Errorf("AccessExpiresIn = %d", pair.AccessExpiresIn)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	repo := &fakeUserRepo{
		getByEmailFn: func(ctx context.Context, email string) (*user.User, error) {
			return &user.User{ID: 7, Email: email, Password: hashPassword(t, "correct-password")}, nil
		},
	}
	svc := newTestService(repo, newFakeTokenStore(), &fakeIssuer{})

	_, err := svc.Login(context.Background(), LoginDTO{Email: "ann@example.com", Password: "wrong-password"})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeUnauthorized {
		t.Fatalf("Login() error = %v, want Unauthorized", err)
	}
}

func TestLoginUnknownUserDoesNotLeakNotFound(t *testing.T) {
	repo := &fakeUserRepo{
		getByEmailFn: func(ctx context.Context, email string) (*user.User, error) {
			return nil, apperror.NotFound("user not found")
		},
	}
	svc := newTestService(repo, newFakeTokenStore(), &fakeIssuer{})

	_, err := svc.Login(context.Background(), LoginDTO{Email: "ghost@example.com", Password: "whatever"})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeUnauthorized {
		t.Fatalf("Login() error = %v, want Unauthorized (not NotFound, to avoid leaking account existence)", err)
	}
}

func TestLoginIssuerError(t *testing.T) {
	repo := &fakeUserRepo{
		getByEmailFn: func(ctx context.Context, email string) (*user.User, error) {
			return &user.User{ID: 7, Email: email, Password: hashPassword(t, "correct-password")}, nil
		},
	}
	svc := newTestService(repo, newFakeTokenStore(), &fakeIssuer{issueErr: errors.New("signing failed")})

	_, err := svc.Login(context.Background(), LoginDTO{Email: "ann@example.com", Password: "correct-password"})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeInternal {
		t.Fatalf("Login() error = %v, want Internal", err)
	}
}

func TestRefreshSuccess(t *testing.T) {
	store := newFakeTokenStore()
	store.tokens["old-refresh"] = 9
	svc := newTestService(&fakeUserRepo{}, store, &fakeIssuer{})

	pair, err := svc.Refresh(context.Background(), "old-refresh")
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if len(store.deleted) != 1 || store.deleted[0] != "old-refresh" {
		t.Errorf("expected old refresh token to be deleted, deleted = %v", store.deleted)
	}
	if pair.RefreshToken == "old-refresh" {
		t.Errorf("expected a newly rotated refresh token")
	}
	if store.tokens[pair.RefreshToken] != 9 {
		t.Errorf("new refresh token not stored against original user id")
	}
}

func TestRefreshInvalidToken(t *testing.T) {
	svc := newTestService(&fakeUserRepo{}, newFakeTokenStore(), &fakeIssuer{})

	_, err := svc.Refresh(context.Background(), "unknown-token")

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeUnauthorized {
		t.Fatalf("Refresh() error = %v, want Unauthorized", err)
	}
}

func TestLogoutDeletesToken(t *testing.T) {
	store := newFakeTokenStore()
	store.tokens["refresh"] = 1
	svc := newTestService(&fakeUserRepo{}, store, &fakeIssuer{})

	if err := svc.Logout(context.Background(), "refresh"); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, ok := store.tokens["refresh"]; ok {
		t.Errorf("expected refresh token to be removed from store")
	}
}

func TestLogoutPropagatesStoreError(t *testing.T) {
	store := newFakeTokenStore()
	store.delErr = errors.New("redis unavailable")
	svc := newTestService(&fakeUserRepo{}, store, &fakeIssuer{})

	err := svc.Logout(context.Background(), "refresh")
	if !errors.Is(err, store.delErr) {
		t.Fatalf("Logout() error = %v, want %v", err, store.delErr)
	}
}
