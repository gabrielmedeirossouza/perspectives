# Sandbox: finance

## Product

Ledger: a personal finance web app where a member tracks accounts, transactions, and budgets. A member signs up with an email and a password and signs in to see only their own data.

## Bootstrap answers

- perspectives: business (from business), member (from user), operator (from operator)
- paths: specs, src, src/specs
- skills: all
- remote: https://github.com/gabrielmedeirossouza/perspectives.git
- history: worktree, never reset

## Decisions

What the person answers when a skill interviews.

- First feature: authentication — sign up with email and password, sign in, sign out, a session that keeps the member signed in.
- One account per email; the email is not verified in this version (out of scope, parked).
- Password: at least 8 characters, no other rule; stored only as a hash; a wrong email or a wrong password answers the same generic message.
- Session: ends on sign out and after 7 days without a request; a signed-out member who opens a member page lands on sign in.
- Surface: a sign-up page, a sign-in page, a sign-out action in the header; field errors inline, the generic message above the form.
- Operator: nothing in this feature; no operator rule yet.
- Stack, when `/sdd-design` asks: frontend React 18 with TypeScript on Vite under `src/web/`; backend Node.js 20 with TypeScript on Fastify under `src/api/`; persistence SQLite through `better-sqlite3`; passwords hashed with `argon2`; tests with Vitest; package manager pnpm, one workspace per app; root manifests under `src/`.
