# identity-service

Authentication, user management and permission service for GoCast.
Gin HTTP API + gRPC permission service, JWT (access/refresh) auth, Casbin RBAC.

## Functionality

- JWT authentication with **rotating single-use refresh tokens** (Redis allow/deny store, reuse detection);
- User CRUD + self profile;
- Email verification (one-time Redis tokens, `email_verified` JWT claim, `POST /auth/verify` / `/auth/resend`);
- RBAC roles (Admin/Member) via Casbin, enforced on gRPC and mirrored to HTTP authorization;
- Bootstrap admin promotion from `ADMIN_EMAIL` at startup (admin is auto email-verified);
- Case-insensitive email normalization on register/login/update;
- Rate limiting on auth endpoints (`login`/`refresh`/`verify`/`resend`);
- Activity events (`user.registered`/`user.login`/`user.email.verified`) enqueued for events-service;
- gRPC permission service (mTLS, OU allow-list + mutation guard) consumed by other services;
- Prometheus `/metrics`.

## Ports

| Protocol | Addr |
|---|---|
| HTTP API (incl. `/metrics`) | `:8080` (`SERVER_ADDR`) |
| gRPC (mTLS) | `:50051` |

## API endpoints

Routes are defined in `internal/delivery/http/routes/routes.go`.

`login`, `refresh`, `verify` and `resend` are rate limited (`AUTH_RATE_LIMIT` per minute per client
IP). Note that public registration `POST /auth/users` is **not** rate limited.

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/auth/login` | – | Login (email + password) |
| `POST` | `/auth/refresh` | refresh cookie | Rotate refresh token, issue new access token |
| `POST` | `/auth/verify` | – | Consume an email-verification token |
| `POST` | `/auth/resend` | bearer | Issue a fresh verification token (revokes the previous one) |
| `POST` | `/auth/users` | – | Public registration (creates a member) |
| `GET` | `/auth/who` | bearer | Current user profile |
| `POST` | `/auth/logout` | bearer + refresh cookie | Logout (deny-list the refresh token) |
| `POST` | `/auth/logout-all` | bearer | Logout from all devices (bumps `token_version`) |
| `GET` | `/auth/users` | bearer + `users/read` | List users (paged, max 100) |
| `GET` | `/auth/users/:id` | bearer + `users/read` | Single user |
| `PATCH` | `/auth/users/:id` | bearer + `users/write` | Update user / role |
| `DELETE` | `/auth/users/:id` | bearer + `users/delete` | Delete user |
| `GET` | `/auth/public-key` | – | JWT public key for consumers |
| `GET` | `/auth/health` | – | Liveness/DB check |
| `GET` | `/metrics` | – | Prometheus metrics |

## Refresh tokens & sessions

Refresh tokens are **single-use and rotated** on every refresh. State lives in Redis (DB 0) as
`sha256(token)` keys:

- **login** → `Allow` (`refresh_allow:<sha256>`, TTL = refresh expiry). Best-effort, but logged
  loudly on failure: without the allow entry the very first refresh would look like a replay;
- **refresh** → `Grant`: the deny list is checked first, then the allow key is consumed atomically
  via a Lua script (`GET` + `DEL` in one round trip) and a new allow key is written;
- **logout** → `Deny` (`refresh_deny:<sha256>`), so the token can never be revived;
- **replay** (already-consumed / mismatched / Redis unavailable) → fail closed: the user's
  `token_version` is bumped, which invalidates **all of their refresh tokens**, and `401` is
  returned. A token denied via logout returns `401 refresh token revoked` without the bump.

The refresh token is only ever set as an `HttpOnly`/`Secure` cookie (never in the response body).
Claims:

- **access**: `user_id`, `role`, `email`, `email_verified`, `iss`, `iat`, `exp` (RS256, short-lived);
- **refresh**: `user_id`, `token_version`, `iss`, `iat`, `exp` — `token_version` is compared against
  the `users` row on every refresh, which is what makes session revocation work.

## Email verification

Registration issues a one-time token stored in Redis as `verify:<token>` → userID, with a
`verify_user:<userID>` index so a resend revokes the previous token. TTL is `VERIFY_TOKEN_TTL`
(default `2h`). On success the `email_verified` claim is set in the user's profile and a fresh
token pair is issued so no re-login is needed.

The mail itself is delivered asynchronously by **mailer-service** (asynq queue `emails`,
Redis DB `REDIS_QUEUE_DB`); enqueueing is best-effort — a Redis failure is logged as a warning and
the request still succeeds. A verification email is also written to the identity-service log
(delivery fallback).

## Activity events

`user.registered`, `user.login` and `user.email.verified` are pushed to the **events-service** asynq
queue (`event:activity`, Redis DB `EVENTS_QUEUE_DB`) for the activity feed. Failures are
best-effort and never fail the originating request.

## RBAC (Casbin)

Model: RBAC (`g = _, _`) + `keyMatch2` on objects, enforced via
`middleware.Authorize(permissionClient, obj, act)`.

- **Admin**: `users read`, `users/* read|write|delete`, `stream read`, `stream/* read|write|delete`
  (explicit, does not inherit member);
- **Member**: `stream read|write`;

per-object policies `users/<id>`/`stream/<id>` keep per-owner record-level access.

Policies live in the Postgres `casbin_rule` table (gorm-adapter). Redis is used as a
**Casbin watcher** (pub/sub channel `/casbin`) so replicas reload policies on changes.
`bootstrapRBAC` runs on start: seeds the role policies above, assigns the `member` role to every
existing user, strips the legacy collection policies (`users read`, `stream read|write`) and
promotes `ADMIN_EMAIL` (email normalized) to admin — auto-verifying that account's email. It uses
the local Casbin enforcer rather than the gRPC client, because the gRPC server is not serving yet
at that point.

## Email normalization

Emails are canonicalized with `strings.ToLower(strings.TrimSpace(...))` on write (register/
update), login lookup and repo defense layer; the DB enforces it with a functional unique index
`idx_users_email_lower = lower(email)`. The index is created by migration `0001_baseline`
in `services/db-migrate`.

## gRPC

The permission API is mirrored over gRPC on `:50051` with **mTLS**: servers require a client cert,
and `AllowOUsInterceptor` rejects peers whose OU is not in `GRPC_TLS_ALLOWED_OUS`
(identity-service, stream-service). Clients use the same CA.

`MutationGuard` is always the first interceptor in the chain (independent of `GRPC_TLS_ENABLED`):
mutating RPCs (`AddPolicy`, `RemovePolicy`, `AddRoleForUser`, `RemoveRoleForUser`) are rejected
unless the peer OU is in the allow-list, so a leaked cert of a data-plane service (thumbnail,
transcoder) cannot escalate privileges. Read RPCs are unrestricted beyond the OU check.

## Configuration (env)

| Variable | Purpose | Default |
|---|---|---|
| `SERVER_ADDR` | HTTP listen addr | `:8080` |
| `DB_HOST` / `DB_PORT` / `DB_NAME` | Postgres | `postgresql`/`5432`/`database1` |
| `DB_USER` / `DB_PASS` | Postgres credentials | _secret_ |
| `REDIS_ADDR` / `REDIS_PASS` | Redis host / password | `localhost` / `` |
| `REDIS_QUEUE_DB` | DB for the mailer `emails` asynq queue | `2` |
| `EVENTS_QUEUE_DB` | DB for the events-service `event:activity` queue | `3` |
| `JWT_SECRET` | HS key (fallback path) | – |
| `JWT_ACCESS_PRIVATE_KEY` / `JWT_ACCESS_PUBLIC_KEY` | RSA key pair for access JWT | _secret_ |
| `JWT_REFRESH_PRIVATE_KEY` / `JWT_REFRESH_PUBLIC_KEY` | RSA key pair for refresh JWT | _secret_ |
| `JWT_ACCESS_TOKEN_EXPIRY` | e.g. `15m` | `15m` |
| `JWT_REFRESH_TOKEN_EXPIRY` | e.g. `168h` | `168h` |
| `JWT_ISSUER` | `iss` claim | `auth-service` |
| `VERIFY_TOKEN_TTL` | Email-verification token lifetime | `2h` |
| `AUTH_RATE_LIMIT` | Rate limit per minute for `login`/`refresh`/`verify`/`resend` | `30` |
| `ADMIN_EMAIL` | Bootstrap admin email | – |
| `AUTH_SERVICE_ADDRESS` | Own gRPC address (client dial target) | – |
| `CASBIN_MODEL` | Path to the Casbin model.conf | – |
| `DOMAIN` | Frontend domain (cookies, links) | – |
| `CORS_ALLOW_ORIGINS` | Comma-separated allowed origins for CORS | `http://localhost:5173,https://example.com,https://api.example.com` |
| `GRPC_TLS_CERT` / `GRPC_TLS_KEY` / `GRPC_TLS_CA` | mTLS client/server certs | _secret_ |
| `GRPC_TLS_ALLOWED_OUS` | Comma-separated allowed peer OUs | – |
| `GRPC_TLS_ENABLED` | Enable mTLS on gRPC | `false` |

Redis is used for four distinct things: the **Casbin watcher** (pub/sub, no DB selection), the
**refresh allow/deny store** and the **verification tokens** (DB 0, `NewRefreshTokenStore` /
`VerificationService` default), the **mailer** queue (DB `REDIS_QUEUE_DB`) and the **events**
queue (DB `EVENTS_QUEUE_DB`).

In K8s the values come from ConfigMap `identity-service-config` (envFrom) + Secret
`go-app-secret` — generated from the root `.env` by `scripts/render-env.sh`.

## Database

Schema is **owned by migrations**, not GORM AutoMigrate:

```bash
# from the monorepo root
make -C services/db-migrate all   # rebuild runner image
make apply-db-migrate             # runs pending identity migrations (Job db-migrate-identity)
```

Version table: `schema_migrations_identity`. Migrations owned by this service:

- `0001_baseline` — `users` (incl. `token_version` and the functional unique index
  `idx_users_email_lower = lower(email)`) + `casbin_rule` + `uuid-ossp`;
- `0002_email_verified` — `ADD/DROP COLUMN IF EXISTS email_verified boolean NOT NULL DEFAULT false`.

See `services/db-migrate/README.md`.

## Build / run

```bash
make build        # Docker image xomrkob/identity-service:<git-tag>
make push         # push to Docker Hub
make deploy       # kubectl set image + rollout

go run cmd/main.go   # local (needs env)
```

## Deploy structure

```
deploy/k8s/
├── base/                  # deployment (envFrom configMap + secrets), service, ingress-class, secret
├── cert-manger/           # local-ca issuer + CA + service certs (cert-manager)
├── grpc-mtls/             # gRPC mTLS certs for identity/stream/thumbnail/transcoder
├── postgres/values.yaml   # bitnami postgres Helm values
├── scaling/hpa.yaml       # CPU/mem autoscaling
└── tests/test-job.yml     # ad-hoc test job
```

HPA: identity scales on CPU/mem (metrics-server based).