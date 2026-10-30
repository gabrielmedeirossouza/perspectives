import { DatabaseSync } from "node:sqlite";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { FastifyInstance } from "fastify";
import { build } from "../server.js";

const DAY = 24 * 60 * 60 * 1000;
const START = new Date("2026-01-01T00:00:00Z");

let app: FastifyInstance;

beforeEach(() => {
  vi.useFakeTimers({ now: START, toFake: ["Date"] });
  app = build({ db: new DatabaseSync(":memory:") });
});

afterEach(() => vi.useRealTimers());

function advance(ms: number) {
  vi.setSystemTime(new Date(Date.now() + ms));
}

const credentials = { email: "Ann@Example.com", password: "longenough" };

async function signUp(body: Record<string, unknown> = credentials) {
  const response = await app.inject({ method: "POST", url: "/auth/sign-up", payload: body });
  return response;
}

function cookieOf(response: { headers: Record<string, unknown> }): string {
  const header = response.headers["set-cookie"];
  const raw = Array.isArray(header) ? header[0] : String(header);
  return raw.split(";")[0];
}

async function sessionStatus(cookie: string) {
  return (await app.inject({ method: "GET", url: "/auth/session", headers: { cookie } })).statusCode;
}

describe("sign-up", () => {
  it("creates the account, normalizes the email, and starts a session", async () => {
    const response = await signUp();
    expect(response.statusCode).toBe(201);
    expect(response.json()).toEqual({ id: expect.any(String), email: "ann@example.com" });
    const cookie = String(response.headers["set-cookie"]);
    expect(cookie).toMatch(/^ledger_session=[0-9a-f]{64};/);
    expect(cookie).toContain("HttpOnly");
    expect(cookie).toContain("SameSite=Lax");
    expect(cookie).toContain("Path=/");
    expect(cookie).toContain("Secure");
    expect(await sessionStatus(cookieOf(response))).toBe(200);
  });

  it("refuses a second account for the same email, whatever its case", async () => {
    await signUp();
    const response = await signUp({ email: " ANN@example.com", password: "otherpassword" });
    expect(response.statusCode).toBe(409);
    expect(response.json()).toEqual({ error: "email_taken" });
    expect(response.headers["set-cookie"]).toBeUndefined();
  });

  it("gives one account to two simultaneous sign-ups of one email", async () => {
    const [a, b] = await Promise.all([signUp(), signUp({ email: "ann@example.com", password: "anotherpass" })]);
    expect([a.statusCode, b.statusCode].sort()).toEqual([201, 409]);
  });

  it("refuses a short or long password, a malformed or long email, and missing fields", async () => {
    for (const body of [
      { email: "a@b.c", password: "short7!" },
      { email: "a@b.c", password: "x".repeat(129) },
      { email: "nope", password: "longenough" },
      { email: "a@" + "b".repeat(253), password: "longenough" },
      { email: "a@b.c" },
      {},
    ]) {
      const response = await signUp(body);
      expect(response.statusCode).toBe(400);
      expect(response.json()).toEqual({ error: "invalid" });
    }
  });

  it("accepts the longest email and password the rules allow", async () => {
    expect((await signUp({ email: "a@" + "b".repeat(252), password: "x".repeat(128) })).statusCode).toBe(201);
  });
});

describe("sign-in", () => {
  it("answers the account and a session for the right credentials", async () => {
    await signUp();
    const response = await app.inject({ method: "POST", url: "/auth/sign-in", payload: { email: "ann@example.com", password: "longenough" } });
    expect(response.statusCode).toBe(200);
    expect(response.json().email).toBe("ann@example.com");
    expect(String(response.headers["set-cookie"])).toMatch(/^ledger_session=/);
  });

  it("refuses a wrong password and an unknown email alike", async () => {
    await signUp();
    const wrongPassword = await app.inject({ method: "POST", url: "/auth/sign-in", payload: { email: "ann@example.com", password: "wrongpassword" } });
    const unknownEmail = await app.inject({ method: "POST", url: "/auth/sign-in", payload: { email: "nobody@example.com", password: "longenough" } });
    const noBody = await app.inject({ method: "POST", url: "/auth/sign-in", payload: {} });
    for (const response of [wrongPassword, unknownEmail, noBody]) {
      expect(response.statusCode).toBe(401);
      expect(response.json()).toEqual({ error: "invalid_credentials" });
      expect(response.headers["set-cookie"]).toBeUndefined();
    }
  });
});

describe("session", () => {
  it("is absent without a cookie and with an unknown cookie", async () => {
    const none = await app.inject({ method: "GET", url: "/auth/session" });
    const unknown = await app.inject({ method: "GET", url: "/auth/session", headers: { cookie: "ledger_session=" + "0".repeat(64) } });
    for (const response of [none, unknown]) {
      expect(response.statusCode).toBe(401);
      expect(response.json()).toEqual({ error: "no_session" });
    }
  });

  it("ends at sign-out, which also answers 204 without a session", async () => {
    const cookie = cookieOf(await signUp());
    const out = await app.inject({ method: "POST", url: "/auth/sign-out", headers: { cookie } });
    expect(out.statusCode).toBe(204);
    expect(String(out.headers["set-cookie"])).toMatch(/^ledger_session=;/);
    expect(await sessionStatus(cookie)).toBe(401);
    expect((await app.inject({ method: "POST", url: "/auth/sign-out" })).statusCode).toBe(204);
  });

  it("ends after 7 days without a request, and slides while requests come", async () => {
    const cookie = cookieOf(await signUp());
    advance(6 * DAY);
    expect(await sessionStatus(cookie)).toBe(200);
    advance(6 * DAY);
    expect(await sessionStatus(cookie)).toBe(200);
    advance(7 * DAY + 1);
    const expired = await app.inject({ method: "GET", url: "/auth/session", headers: { cookie } });
    expect(expired.statusCode).toBe(401);
    expect(expired.json()).toEqual({ error: "no_session" });
    advance(-7 * DAY);
    expect(await sessionStatus(cookie)).toBe(401);
  });

  it("answers only the account the session belongs to", async () => {
    const ann = cookieOf(await signUp());
    const bob = cookieOf(await signUp({ email: "bob@example.com", password: "longenough" }));
    expect((await app.inject({ method: "GET", url: "/auth/session", headers: { cookie: ann } })).json().email).toBe("ann@example.com");
    expect((await app.inject({ method: "GET", url: "/auth/session", headers: { cookie: bob } })).json().email).toBe("bob@example.com");
  });
});
