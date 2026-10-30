import { DatabaseSync } from "node:sqlite";
import { describe, expect, it } from "vitest";
import { build } from "./server.js";

function app() {
  return build({ db: new DatabaseSync(":memory:") });
}

describe("API error codes", () => {
  it("answers not_found for a route that does not exist", async () => {
    const response = await app().inject({ method: "GET", url: "/nowhere" });
    expect(response.statusCode).toBe(404);
    expect(response.json()).toEqual({ error: "not_found" });
  });

  it("answers invalid for a body that cannot be parsed", async () => {
    const api = app();
    for (const request of [
      { payload: "{not json", headers: { "content-type": "application/json" } },
      { payload: "", headers: { "content-type": "application/json" } },
      { payload: "email=a@b.c", headers: { "content-type": "text/plain" } },
    ]) {
      const response = await api.inject({ method: "POST", url: "/auth/sign-up", ...request });
      expect(response.statusCode).toBe(400);
      expect(response.json()).toEqual({ error: "invalid" });
    }
  });

  it("answers internal_error for an unhandled exception, with no other field", async () => {
    const api = app();
    api.get("/boom", async () => {
      throw new Error("secret detail");
    });
    const response = await api.inject({ method: "GET", url: "/boom" });
    expect(response.statusCode).toBe(500);
    expect(response.json()).toEqual({ error: "internal_error" });
  });
});
