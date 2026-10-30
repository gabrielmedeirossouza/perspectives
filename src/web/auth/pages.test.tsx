import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, useRoutes } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { routes } from "../routes";
import { EMAIL_TAKEN } from "./SignUpPage";
import { INVALID_CREDENTIALS } from "./SignInPage";

type Reply = { status: number; body?: unknown };

function stubFetch(replies: Record<string, Reply>) {
  const calls: { url: string; init?: RequestInit }[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url: string, init?: RequestInit) => {
      calls.push({ url, init });
      const reply = replies[url] ?? { status: 500, body: { error: "unstubbed" } };
      return { ok: reply.status < 400, status: reply.status, json: async () => reply.body };
    }),
  );
  return calls;
}

/** The app's own route table, in a memory router. */
function App() {
  return useRoutes(routes);
}

function renderAt(path: string) {
  render(
    <MemoryRouter initialEntries={[path]}>
      <App />
    </MemoryRouter>,
  );
}

function fill(email: string, password: string) {
  fireEvent.change(screen.getByLabelText("Email"), { target: { value: email } });
  fireEvent.change(screen.getByLabelText("Password"), { target: { value: password } });
}

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("sign-up page", () => {
  it("shows field errors and sends nothing for an empty field or a short password", async () => {
    const calls = stubFetch({});
    renderAt("/sign-up");
    fireEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect(await screen.findByText("Enter your email")).toBeTruthy();
    fill("ann@example.com", "short");
    fireEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect(await screen.findByText("Use at least 8 characters")).toBeTruthy();
    expect(calls).toHaveLength(0);
  });

  it("shows the taken-email message above the form", async () => {
    stubFetch({ "/auth/sign-up": { status: 409, body: { error: "email_taken" } } });
    renderAt("/sign-up");
    fill("ann@example.com", "longenough");
    fireEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect((await screen.findByRole("alert")).textContent).toBe(EMAIL_TAKEN);
  });

  it("opens the home page, signed in, after a successful sign-up", async () => {
    const calls = stubFetch({
      "/auth/sign-up": { status: 201, body: { id: "1", email: "ann@example.com" } },
      "/auth/session": { status: 200, body: { id: "1", email: "ann@example.com" } },
    });
    renderAt("/sign-up");
    fill("ann@example.com", "longenough");
    fireEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect(await screen.findByRole("heading", { name: "Home" })).toBeTruthy();
    expect(calls[0].init?.credentials).toBe("include");
    expect(JSON.parse(String(calls[0].init?.body))).toEqual({ email: "ann@example.com", password: "longenough" });
  });
});

describe("sign-in page", () => {
  it("shows the same generic message for refused credentials", async () => {
    stubFetch({ "/auth/sign-in": { status: 401, body: { error: "invalid_credentials" } } });
    renderAt("/sign-in");
    fill("ann@example.com", "whatever");
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
    expect((await screen.findByRole("alert")).textContent).toBe(INVALID_CREDENTIALS);
  });

  it("sends nothing for an empty field", async () => {
    const calls = stubFetch({});
    renderAt("/sign-in");
    fill("ann@example.com", "");
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
    expect(await screen.findByText("Enter your password")).toBeTruthy();
    expect(calls).toHaveLength(0);
  });

  it("opens the home page after a successful sign-in", async () => {
    stubFetch({
      "/auth/sign-in": { status: 200, body: { id: "1", email: "ann@example.com" } },
      "/auth/session": { status: 200, body: { id: "1", email: "ann@example.com" } },
    });
    renderAt("/sign-in");
    fill("ann@example.com", "longenough");
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
    expect(await screen.findByRole("heading", { name: "Home" })).toBeTruthy();
  });
});

describe("session guard", () => {
  it("opens the sign-in page for a member page without a session", async () => {
    stubFetch({ "/auth/session": { status: 401, body: { error: "no_session" } } });
    renderAt("/");
    expect(await screen.findByRole("heading", { name: "Sign in" })).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "Home" })).toBeNull();
  });

  it("shows the page under a header whose sign-out ends the session and opens sign-in", async () => {
    const calls = stubFetch({
      "/auth/session": { status: 200, body: { id: "1", email: "ann@example.com" } },
      "/auth/sign-out": { status: 204 },
    });
    renderAt("/");
    expect(await screen.findByRole("heading", { name: "Home" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Sign out" }));
    expect(await screen.findByRole("heading", { name: "Sign in" })).toBeTruthy();
    expect(calls.map((c) => c.url)).toContain("/auth/sign-out");
  });
});
