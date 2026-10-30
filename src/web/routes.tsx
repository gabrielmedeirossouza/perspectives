import type { RouteObject } from "react-router";
import { RequireSession } from "./auth/RequireSession";
import { SignInPage } from "./auth/SignInPage";
import { SignUpPage } from "./auth/SignUpPage";

/** The home page has no spec yet (specs/TODO.md): a placeholder behind the session guard. */
function Home() {
  return (
    <main>
      <h1>Home</h1>
    </main>
  );
}

export const routes: RouteObject[] = [
  { path: "/sign-up", element: <SignUpPage /> },
  { path: "/sign-in", element: <SignInPage /> },
  { element: <RequireSession />, children: [{ path: "/", element: <Home /> }] },
];
