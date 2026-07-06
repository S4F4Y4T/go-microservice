# Testing

## The Problem

The repo had zero `_test.go` files across `pkg/` and every service. `make test` (`go test ./...`) ran but had nothing to verify — a passing test suite was really just "nothing to report." Business logic (token issuance, password hashing, error-code mapping, request validation) had no regression protection: a refactor could silently invert an `Unauthorized`/`NotFound` check and nothing would fail.

## Strategy: Test the Pure Layers First

Not all code is equally cheap to test. The repo naturally splits into three layers of testability:

| Layer | Example | Testability |
|---|---|---|
| Pure logic | `pkg/pagination`, `pkg/apperror`, `pkg/query` | Trivial — no I/O, plain functions |
| Logic behind an interface | `services/auth/internal/auth.Service` (depends on `UserLookup`, `token.Store`, `token.AccessIssuer`) | Easy — swap the interface for an in-memory fake |
| Thin I/O wiring | `RedisTokenRepository`, `internal/router`, `cmd/api/main.go` | Hard — needs a real Redis/Postgres or an integration harness |

Coverage was built in that order: shared `pkg/` first, then the `auth` service's `Service`/`Handler`, deferring the I/O-wiring layer. This front-loads the highest ratio of confidence-gained to effort-spent, and every layer above depends on the layers below being correct.

## `pkg/` — Shared Library Tests

Each package gets a `_test.go` file next to the code it covers, using table-driven tests over the standard `testing` package (no assertion library — the codebase doesn't already depend on one, and stdlib comparisons are enough for these cases):

- **`apperror`** — constructor → `Code`/`HTTPStatus` mapping, `From()` normalization (already-`*AppError` passes through, wrapped `AppError` unwraps via `errors.As`, `gorm.ErrRecordNotFound` maps to `NotFound`, unknown errors become `Internal` with the cause preserved).
- **`pagination`** — page/limit clamping (`<1` defaults, `>100` clamps), `Offset()` arithmetic, `NewMeta()` total-pages rounding including the zero-limit divide-by-zero guard.
- **`query`** — sort/filter parsing against an allowlist `Schema`: unknown fields dropped, non-sortable/non-filterable fields dropped, malformed `filter[x]` keys ignored, combined sort+filter parsing.
- **`validation`** — `required`/`email`/`min`/`max` tag messages, and that field names in errors come from the `json` tag, not the Go field name.
- **`token`** — JWT generate/parse round-trip, expired token rejected, wrong public key rejected, **alg-substitution attack rejected** (an HS256-signed token is refused because `ParseUserID` only accepts `*jwt.SigningMethodRSA`), malformed token string rejected.
- **`response`** — `Success`/`SuccessWithMeta`/`NoContent` write the expected envelope; `Error()` maps an `*AppError` to its HTTP status and, critically, confirms an arbitrary internal error's message text never reaches the client body (`db connection dropped` must not appear in the JSON response).
- **`config`** — env var parsing (`GetEnvInt`/`GetEnvDuration`/`GetEnvBool`) falls back to the default on missing *or* malformed values, rather than panicking or zeroing.
- **`logger`** — `ParseLevel` name→`slog.Level` mapping (case-insensitive, whitespace-trimmed, unknown → `Info`), and `WithContext`/`FromContext` round-trip.

Run with:
```
cd pkg && go test ./...
```

## `services/auth` — Business Logic Tests

`auth.Service` is constructed from three interfaces (`UserLookup`, `token.Store`, `token.AccessIssuer`), not concrete Redis/gRPC clients — this was already true before testing started, and it's what makes the service layer testable without a database. `service_test.go` defines three in-memory fakes (`fakeUserRepo`, `fakeTokenStore`, `fakeIssuer`) built from plain structs with optional function fields, so each test only wires up the behavior it needs.

Covered in `service_test.go`:
- **`Register`** — success (and that the stored password is actually bcrypt-hashed, not the plaintext), duplicate email → `Conflict`, repository error propagation.
- **`Login`** — success issues an access token and stores a refresh token against the right user ID; wrong password → `Unauthorized`; an unknown email also surfaces `Unauthorized` rather than `NotFound` (**this is deliberate — leaking "no such user" vs "wrong password" lets an attacker enumerate registered emails**); token-issuer failure → `Internal`.
- **`Refresh`** — rotates the refresh token (old one deleted, new one stored against the same user ID); an unrecognized refresh token → `Unauthorized`.
- **`Logout`** — deletes the refresh token; store error propagates to the caller.

Covered in `http_handler_test.go` (using `httptest.NewRecorder`/`httptest.NewRequest`, no real server):
- Register: 201 + success envelope, 400 on validation failure, 409 on duplicate email.
- Login: sets an `HttpOnly` refresh cookie on success, 401 on bad credentials.
- Refresh: 401 when the cookie is missing, rotates the cookie value on success.
- Logout: always clears the cookie (`MaxAge < 0`) even if no refresh cookie was present.

Run with:
```
cd services/auth && go test ./...
```

## Fakes Over Mocking Frameworks

No `gomock`/`mockery`-generated mocks — the interfaces here are small (2–3 methods) and the behavior needed per test varies (return this error, capture this argument), which hand-written fakes with optional func fields express directly:

```go
type fakeUserRepo struct {
    existsByEmailFn func(ctx context.Context, email string) (bool, error)
    getByEmailFn    func(ctx context.Context, email string) (*user.User, error)
    createFn        func(ctx context.Context, u *user.User) (*user.User, error)
}
```
A test only sets the field it cares about; unset fields fall back to a sane default (e.g. `ExistsByEmail` defaults to `false, nil`). This avoids a mock-generation build step and keeps the fakes co-located with the tests that use them.

## What's Not Covered Yet

- **`RedisTokenRepository`** (`services/auth/internal/auth/repository.go`) — needs a real Redis or `miniredis`; the interface it implements (`token.Store`) is already covered indirectly through `fakeTokenStore` in the service tests, but the Redis-specific key-prefixing and `redis.Nil` handling is not.
- **`services/user`, `services/notification`** business logic — same pattern as `auth` should apply once those `Service` layers are reviewed for interface seams.
- **`internal/router`, `internal/health`, `cmd/api/main.go`** across all services — thin wiring, best covered by a small number of integration tests rather than unit tests.
- **`pkg/middleware`, `pkg/grpcmiddleware`, `pkg/request`, `pkg/mailer`, `pkg/messaging/rabbitmq`** — `pkg/request.DecodeJSON` is pure enough to unit test with `httptest` next; the middleware and mailer/rabbitmq packages wrap `net/http`/SMTP/AMQP and are better suited to either an `httptest.Server` chain test or a `docker-compose`-backed integration test.
- **CI** — none of this runs automatically yet; there's no `.github/workflows` (or equivalent) invoking `make test` on push.

## Alternatives Considered

- **testify (`assert`/`require`)** — already present as an indirect dependency (via `services/notification`), so promoting it to a direct dependency across modules was an option. Skipped for now: the assertions needed here are simple equality/error-type checks that stdlib `if`/`t.Errorf` express just as clearly, and it avoids adding a new direct dependency to modules that don't have one yet. Worth reconsidering if table-driven tests grow complex enough that `assert.Equal`'s diff output starts paying for itself.
- **Mock generation (`gomock`, `mockery`)** — rejected for the reasons above (small interfaces, per-test behavior variance); revisit if an interface grows large enough that hand-written fakes become repetitive.
- **Integration tests against real Redis/Postgres via `docker-compose`** — the right tool for `RedisTokenRepository` and the GORM repositories, but out of scope for this pass, which focused on business logic that doesn't require external services at all.
