import { useEffect, useState } from "react";
import { Navigate, Outlet } from "react-router";
import { session, type Account } from "./api";
import { SignOutButton } from "./SignOutButton";

type State = { kind: "loading" } | { kind: "signed-out" } | { kind: "signed-in"; account: Account };

/** Renders member pages under a header with sign-out; without a live session, opens sign-in. */
export function RequireSession() {
  const [state, setState] = useState<State>({ kind: "loading" });
  useEffect(() => {
    let active = true;
    session().then((result) => {
      if (!active) return;
      setState(result.ok ? { kind: "signed-in", account: result.account } : { kind: "signed-out" });
    });
    return () => {
      active = false;
    };
  }, []);
  if (state.kind === "loading") return null;
  if (state.kind === "signed-out") return <Navigate to="/sign-in" replace />;
  return (
    <>
      <header>
        <span>{state.account.email}</span>
        <SignOutButton />
      </header>
      <Outlet />
    </>
  );
}
