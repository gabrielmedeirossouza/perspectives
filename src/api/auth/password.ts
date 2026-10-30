import { randomBytes, scryptSync, timingSafeEqual } from "node:crypto";

const N = 16384;
const R = 8;
const P = 1;
const KEY_LENGTH = 64;

/** Hashes a password as scrypt$<salt hex>$<key hex>. */
export function hashPassword(password: string): string {
  const salt = randomBytes(16);
  const key = scryptSync(password, salt, KEY_LENGTH, { N, r: R, p: P });
  return `scrypt$${salt.toString("hex")}$${key.toString("hex")}`;
}

/** Compares a password with a stored hash in constant time. */
export function verifyPassword(password: string, stored: string): boolean {
  const [scheme, saltHex, keyHex] = stored.split("$");
  if (scheme !== "scrypt" || !saltHex || !keyHex) return false;
  const key = scryptSync(password, Buffer.from(saltHex, "hex"), KEY_LENGTH, { N, r: R, p: P });
  const expected = Buffer.from(keyHex, "hex");
  return key.length === expected.length && timingSafeEqual(key, expected);
}

/** A hash of a throwaway password, compared when no account matches so a wrong email costs a hash too. */
export const DECOY_HASH = hashPassword(randomBytes(16).toString("hex"));
