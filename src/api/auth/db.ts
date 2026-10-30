import type { DatabaseSync } from "node:sqlite";
import { randomBytes, randomUUID } from "node:crypto";

export interface Account {
  id: string;
  email: string;
  password_hash: string;
}

export interface Session {
  id: string;
  account_id: string;
  last_seen_at: string;
}

/** Creates the account and session tables when absent. */
export function migrate(db: DatabaseSync): void {
  db.exec(`
    CREATE TABLE IF NOT EXISTS account (
      id TEXT PRIMARY KEY,
      email TEXT NOT NULL UNIQUE,
      password_hash TEXT NOT NULL,
      created_at TEXT NOT NULL
    );
    CREATE TABLE IF NOT EXISTS session (
      id TEXT PRIMARY KEY,
      account_id TEXT NOT NULL REFERENCES account(id),
      last_seen_at TEXT NOT NULL
    );
  `);
}

/** Emails are stored lowercase and trimmed. */
export function normalizeEmail(email: string): string {
  return email.trim().toLowerCase();
}

export function findAccountByEmail(db: DatabaseSync, email: string): Account | undefined {
  return db.prepare("SELECT id, email, password_hash FROM account WHERE email = ?").get(normalizeEmail(email)) as Account | undefined;
}

export function findAccountById(db: DatabaseSync, id: string): Account | undefined {
  return db.prepare("SELECT id, email, password_hash FROM account WHERE id = ?").get(id) as Account | undefined;
}

/** Inserts an account; returns undefined when the email already has one. */
export function createAccount(db: DatabaseSync, email: string, passwordHash: string, now: Date): Account | undefined {
  const account: Account = { id: randomUUID(), email: normalizeEmail(email), password_hash: passwordHash };
  try {
    db.prepare("INSERT INTO account (id, email, password_hash, created_at) VALUES (?, ?, ?, ?)").run(account.id, account.email, account.password_hash, now.toISOString());
  } catch (error) {
    if (error instanceof Error && error.message.includes("UNIQUE")) return undefined;
    throw error;
  }
  return account;
}

export function createSession(db: DatabaseSync, accountId: string, now: Date): Session {
  const session: Session = { id: randomBytes(32).toString("hex"), account_id: accountId, last_seen_at: now.toISOString() };
  db.prepare("INSERT INTO session (id, account_id, last_seen_at) VALUES (?, ?, ?)").run(session.id, session.account_id, session.last_seen_at);
  return session;
}

export function findSession(db: DatabaseSync, id: string): Session | undefined {
  return db.prepare("SELECT id, account_id, last_seen_at FROM session WHERE id = ?").get(id) as Session | undefined;
}

export function touchSession(db: DatabaseSync, id: string, now: Date): void {
  db.prepare("UPDATE session SET last_seen_at = ? WHERE id = ?").run(now.toISOString(), id);
}

export function deleteSession(db: DatabaseSync, id: string): void {
  db.prepare("DELETE FROM session WHERE id = ?").run(id);
}
