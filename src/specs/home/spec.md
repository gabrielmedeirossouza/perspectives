# Home

How the home page is built: placement and route.

## Placement

### Home page of the web app

The home page **must** live under `src/web/home/` as `HomePage.tsx`. Serves [Home content](../../../specs/member/home/spec.md#home-content).

### Home route registration

`src/web/routes.tsx` **must** register the home page at `/` as a child of the session guard of `src/web/auth/`, so a request without a session never renders it. Serves [Home route](../../../specs/member/home/spec.md#home-route).
