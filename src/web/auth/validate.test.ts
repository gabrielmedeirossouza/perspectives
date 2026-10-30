import { describe, expect, it } from "vitest";
import { validateSignIn, validateSignUp } from "./validate";

describe("validate", () => {
  it("names the empty fields of a sign-in", () => {
    expect(validateSignIn("", "")).toEqual({ email: "Enter your email", password: "Enter your password" });
    expect(validateSignIn("a@b.c", "x")).toEqual({});
  });

  it("adds the minimum length to a sign-up", () => {
    expect(validateSignUp("a@b.c", "short")).toEqual({ password: "Use at least 8 characters" });
    expect(validateSignUp("a@b.c", "")).toEqual({ password: "Enter your password" });
    expect(validateSignUp("a@b.c", "longenough")).toEqual({});
  });
});
