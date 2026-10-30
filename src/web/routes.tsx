import type { RouteObject } from "react-router";
import { RequireSession } from "./auth/RequireSession";
import { SignInPage } from "./auth/SignInPage";
import { SignUpPage } from "./auth/SignUpPage";
import { HomePage } from "./home/HomePage";

export const routes: RouteObject[] = [
  { path: "/sign-up", element: <SignUpPage /> },
  { path: "/sign-in", element: <SignInPage /> },
  { element: <RequireSession />, children: [{ path: "/", element: <HomePage /> }] },
];
