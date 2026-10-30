# Bug autopsies

Stateful record of bugs corrected in `src/`.

## Entry format

Each corrected bug gets one entry using this structure:

### [YYYY-MM-DD] Short bug title

- **Symptom:** What was observed.
- **Impact:** Who or what was affected.
- **Root cause:** The precise technical cause.
- **Why it happened:** Why the existing code and tests allowed the bug.
- **Correction:** What changed to fix it.

## Autopsies

<!-- Add the newest correction at the top. -->

### [2026-10-30] Fastify default error bodies broke the error contract

- **Symptom:** A request to an unknown route, a request whose JSON body could not be parsed, and an unhandled exception answered Fastify's default body `{ statusCode, error, message }` instead of `{ "error": "<code>" }`.
- **Impact:** Any API client that reads `error` as a snake_case code got `"Not Found"`, `"Bad Request"`, or `"Internal Server Error"`. A 500 also returned the exception message.
- **Root cause:** `build()` in `src/api/server.ts` set no not-found handler and no error handler, so every error the routes did not reply to themselves fell back to Fastify's defaults.
- **Why it happened:** The tests only covered the routes' own explicit replies. No test sent a request that Fastify answers before or around a route.
- **Correction:** `build()` now sets a not-found handler that answers `404 not_found`, and an error handler that answers `400 invalid` for a body-parsing error (`FST_ERR_CTP_*`) and `500 internal_error` otherwise. `src/api/server.test.ts` covers the three cases.
