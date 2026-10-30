import { describe, expect, it } from "vitest";
import { hashPassword, verifyPassword } from "./password.js";

describe("password hashing", () => {
  it("stores scrypt with a 16-byte salt and a 64-byte key, and verifies the password", () => {
    const hash = hashPassword("correct horse battery");
    expect(hash).toMatch(/^scrypt\$[0-9a-f]{32}\$[0-9a-f]{128}$/);
    expect(verifyPassword("correct horse battery", hash)).toBe(true);
    expect(verifyPassword("wrong", hash)).toBe(false);
  });

  it("salts every hash", () => {
    expect(hashPassword("same")).not.toBe(hashPassword("same"));
  });

  it("refuses a stored value that is not a scrypt hash", () => {
    expect(verifyPassword("x", "plain")).toBe(false);
    expect(verifyPassword("x", "scrypt$zz$")).toBe(false);
  });
});
