# identity-service

Authentication, user management and permission service for GoCast.
Gin HTTP API + gRPC permission service, JWT (access/refresh) auth, Casbin RBAC.

## Functionality

- JWT authentication with refresh tokens;
- User CRUD + self profile;
- RBAC roles (Admin/Member) via Casbin, enforced on gRPC and mirrored to HTTP authorization;
- Bootstrap admin promotion from `ADMIN_EMAIL` at startup;
- Case-insensitive email normalization on register/login/update;
- gRPC permission service (mTLS, OU allow-list) consumed by other services;
- Prometheus `/metrics`.

## Ports

| Protocol | Addr |
|---|---|
| HTTP API (incl. `/metrics`) | `:8080` (`SERVER_ADDR`) |
| gRPC (mTLS) | `:50051` |

## API endpoints

Routes are defined in `internal/delivery/http/routes/routes.go`.

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/auth/login` | – | Login (email + password) |
| `POST` | `/auth/refresh` | – | Refresh access token |
| `POST` | `/auth/users` | – | Public registration (creates a member) |
| `GET` | `/auth/who` | bearer | Current user profile |
| `POST` | `/auth/logout` | bearer | Logout (revoke current refresh) |
| `POST` | `/auth/logout-all` | bearer | Logout from all devices |
| `GET` | `/auth/users` | bearer + `users/read` | List users (paged, max 100) |
| `GET` | `/auth/users/:id` | bearer + `users/read` | Single user |
| `PATCH` | `/auth/users/:id` | bearer + `users/write` | Update user / role |
| `DELETE` | `/auth/users/:id` | bearer + `users/delete` | Delete user |
| `GET` | `/auth/public-key` | – | JWT public key for consumers |
| `GET` | `/auth/health` | – | Liveness/DB check |
| `GET` | `/metrics` | – | Prometheus metrics |

## RBAC (Casbin)

Model: RBAC (`g = _, _`) + `keyMatch2` on objects, enforced via
`middleware.Authorize(permissionClient, obj, act)`.

- **Admin**: `users read` + `users/* read|write|delete` (explicit, does not inherit member);
- **Member**: `stream read|write`;

per-object policies `users/<id>`/`stream/<id>` keep per-owner record-level access.

Policies live in the Postgres `casbin_rule` table (gorm-adapter). Redis is used as a
**Casbin watcher** (pub/sub channel `/casbin`) so replicas reload policies on changes.
`bootstrapRBAC` runs on start: seeds roles and promotes `ADMIN_EMAIL` (email normalized) to admin.

## Email normalization

Emails are canonicalized with `strings.ToLower(strings.TrimSpace(...))` on write (register/
update), login lookup and repo defense layer; the DB enforces it with a functional unique index
`idx_users_email_lower = lower(email)`. The index is created by migration `0001_baseline`
in `services/db-migrate`.

## gRPC

`/auth` HTTP is mirrored over gRPC on `:50051` with **mTLS**:
servers require a client cert, and `AllowOUsInterceptor` rejects peers whose OU is not in
`GRPC_TLS_ALLOWED_OUS` (identity-service, stream-service). Clients use the same CA.

## Configuration (env)

| Variable | Purpose | Default |
|---|---|---|
| `SERVER_ADDR` | HTTP listen addr | `:8080` |
| `DB_HOST` / `DB_PORT` / `DB_NAME` | Postgres | `postgresql`/`5432`/`database1` |
| `DB_USER` / `DB_PASS` | Postgres credentials | _secret_ |
| `REDIS_ADDR` / `REDIS_PASS` | Casbin watcher pub/sub | `localhost` / `` |
| `JWT_SECRET` | HS key (fallback path) | – |
| `JWT_ACCESS_PRIVATE_KEY` / `JWT_ACCESS_PUBLIC_KEY` | RSA key pair for access JWT | _secret_ |
| `JWT_REFRESH_PRIVATE_KEY` / `JWT_REFRESH_PUBLIC_KEY` | RSA key pair for refresh JWT | _secret_ |
| `JWT_ACCESS_TOKEN_EXPIRY` | e.g. `15m` | `15m` |
| `JWT_REFRESH_TOKEN_EXPIRY` | e.g. `168h` | `168h` |
| `ADMIN_EMAIL` | Bootstrap admin email | – |
| `CASBIN_MODEL` | Path to the Casbin model.conf | – |
| `DOMAIN` | Frontend domain (cookies, links) | – |
| `CORS_ALLOW_ORIGINS` | Comma-separated allowed origins for CORS | `http://localhost:5173,https://example.com,https://api.example.com` |
| `GRPC_TLS_CERT` / `GRPC_TLS_KEY` / `GRPC_TLS_CA` | mTLS client/server certs | _secret_ |
| `GRPC_TLS_ALLOWED_OUS` | Comma-separated allowed peer OUs | – |
| `GRPC_TLS_ENABLED` | Enable mTLS on gRPC | `false` |

In K8s the values come from ConfigMap `identity-service-config` (envFrom) + Secret
`go-app-secret` — generated from the root `.env` by `scripts/render-env.sh`.

## Database

Schema is **owned by migrations**, not GORM AutoMigrate:

```bash
# from the monorepo root
make -C services/db-migrate all   # rebuild runner image
make apply-db-migrate             # runs pending identity migrations (Job db-migrate-identity)
```

Version table: `schema_migrations_identity`. See `services/db-migrate/README.md`.

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