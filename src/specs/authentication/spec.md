# Authentication

How authentication is built: placement, persistence, password hashing, the session cookie, and the HTTP contract.

## Placement

### Auth module of the API

The routes, the account and session store, and the password hashing **must** live under `src/api/auth/`, with `src/api/auth/routes.ts` registering every `/auth` route as a Fastify plugin. Serves [Credentials prove the account](../../../specs/business/authentication/spec.md#credentials-prove-the-account).

### Auth pages of the web app

The sign-up page, the sign-in page, the sign-out action, and the guard that sends a member without a session to sign in **must** live under `src/web/auth/`. Serves [Sign-up page](../../../specs/member/authentication/spec.md#sign-up-page), [Sign-in page](../../../specs/member/authentication/spec.md#sign-in-page), [Sign-out action](../../../specs/member/authentication/spec.md#sign-out-action), and [Signed-out landing](../../../specs/member/authentication/spec.md#signed-out-landing).

## Persistence

### SQLite through node:sqlite

Accounts and sessions **must** be stored in one SQLite database opened with `DatabaseSync` of `node:sqlite`, at the path of the environment variable `LEDGER_DB` (`:memory:` in tests). The table `account` **must** hold `id TEXT PRIMARY KEY`, `email TEXT NOT NULL UNIQUE`, `password_hash TEXT NOT NULL`, `created_at TEXT NOT NULL`; the table `session` **must** hold `id TEXT PRIMARY KEY`, `account_id TEXT NOT NULL REFERENCES account(id)`, `last_seen_at TEXT NOT NULL`. Emails are stored lowercase and trimmed. Serves [One account per email](../../../specs/business/authentication/spec.md#one-account-per-email) and [Session lifetime](../../../specs/business/authentication/spec.md#session-lifetime).

### Password hashing with scrypt

A password **must** be stored only as its `scrypt` hash from `node:crypto` — a 16-byte random salt, `N` 16384, `r` 8, `p` 1, 64 bytes of key — encoded as `scrypt$<salt hex>$<key hex>`, and compared with `timingSafeEqual`. The plain password never leaves the request handler. Serves [Credentials prove the account](../../../specs/business/authentication/spec.md#credentials-prove-the-account).

## Session

### Session cookie

A session **must** be identified by 32 random bytes in hex, carried in the cookie `ledger_session` with `HttpOnly`, `SameSite=Lax`, `Path=/`, and `Secure` unless `NODE_ENV` is `development`. A request with the cookie **must** load the session; one whose `last_seen_at` is more than 7 days before now is deleted and the request refused with `401 { "error": "no_session" }`; otherwise `last_seen_at` is set to now. A route that needs a session reads the account id from it and nothing else identifies the member. Serves [Session lifetime](../../../specs/business/authentication/spec.md#session-lifetime) and [Data isolation by account](../../../specs/business/authentication/spec.md#data-isolation-by-account).

## HTTP

### Sign-up endpoint

`POST /auth/sign-up` with the JSON body `{ "email", "password" }` **must** answer `201 { "id", "email" }` and set the session cookie; `409 { "error": "email_taken" }` when the email already identifies an account; `400 { "error": "invalid" }` when the email has no `@` or the password has fewer than 8 characters. Serves [One account per email](../../../specs/business/authentication/spec.md#one-account-per-email) and [Minimum password length](../../../specs/business/authentication/spec.md#minimum-password-length).

### Sign-in endpoint

`POST /auth/sign-in` with `{ "email", "password" }` **must** answer `200 { "id", "email" }` and set the session cookie, or `401 { "error": "invalid_credentials" }` — the same status and body for a wrong email and for a wrong password, in the same time order of magnitude: a missing account still runs one hash comparison. Serves [Credentials prove the account](../../../specs/business/authentication/spec.md#credentials-prove-the-account).

### Sign-out endpoint

`POST /auth/sign-out` **must** delete the session of the cookie, clear the cookie, and answer `204`, also when no session exists. Serves [Session lifetime](../../../specs/business/authentication/spec.md#session-lifetime).

### Session endpoint

`GET /auth/session` **must** answer `200 { "id", "email" }` for a live session and `401 { "error": "no_session" }` otherwise; the web guard calls it before rendering a member page. Serves [Signed-out landing](../../../specs/member/authentication/spec.md#signed-out-landing).
