# Toolchain

How every part of Ledger is built, placed, and verified: runtime, language, package layout, libraries, and test runner.

## Runtime

### Node.js 26 and TypeScript 5

Every package **must** run on Node.js 26 and be written in TypeScript 5 with `"strict": true`. The API is compiled by `tsc` to `dist/`; the web app is built by Vite. No JavaScript source file is written by hand.

### npm workspaces under src

`src/package.json` **must** declare the npm workspaces `api` and `web`, pin `engines.node` to `>=26`, and be the only place dependencies are installed from (`npm install` in `src/`); `src/package-lock.json` is the only lockfile. The scripts `test`, `typecheck`, and `build` of `src/package.json` **must** run the same script of every workspace.

## Layout

### One directory per app

The API **must** live under `src/api/` and the web app under `src/web/`, each a workspace with its own `package.json` and `tsconfig.json`. Inside an app, code is grouped by feature (`src/api/auth/`), never by technical type (`controllers/`, `services/`).

### Ignored artifacts

The root `.gitignore` **must** ignore `node_modules/`, `dist/`, and `*.db` everywhere under `src/`: installed packages, build output, and local databases are never tracked and never content of a code node.

## Libraries

### Fastify as the HTTP server

The API **must** serve HTTP with `fastify` 5. `src/api/server.ts` **must** export `build(options)` returning the configured Fastify instance, so tests inject requests without a port, and `src/api/main.ts` **must** be the only file that listens: on host `0.0.0.0` and the port of the environment variable `PORT`, 3000 when unset. Every response body is JSON; an error body is `{ "error": "<code>" }` with a snake_case code.

### React on Vite

The web app **must** be React 18 on Vite 7, with `react-router` 7 for routes. `src/web/main.tsx` **must** be the entry; a feature's pages live in its directory and register their routes in `src/web/routes.tsx`. The app calls the API through `fetch` with `credentials: "include"`, on the base URL `VITE_API_URL`.

### Vitest as the test runner

Unit tests **must** run on `vitest` 3, beside the code they cover as `<name>.test.ts` or `<name>.test.tsx`; time-dependent behavior is tested with Vitest's fake timers, never through an injected clock. Web component tests **must** render with `@testing-library/react` under the `jsdom` environment. `npm test` in `src/` **must** run every workspace's suite.
