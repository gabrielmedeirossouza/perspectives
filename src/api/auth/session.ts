import type { FastifyReply, FastifyRequest } from "fastify";
import type { DatabaseSync } from "node:sqlite";
import { createSession, deleteSession, findSession, touchSession, type Session } from "./db.js";

export const COOKIE = "ledger_session";
export const SESSION_DAYS = 7;
const DAY_MS = 24 * 60 * 60 * 1000;

export function cookieOptions() {
  return {
    httpOnly: true,
    sameSite: "lax" as const,
    path: "/",
    secure: process.env.NODE_ENV !== "development",
  };
}

/** Starts a session for the account and sets its cookie on the reply. */
export function startSession(db: DatabaseSync, reply: FastifyReply, accountId: string, now: Date): Session {
  const session = createSession(db, accountId, now);
  reply.setCookie(COOKIE, session.id, cookieOptions());
  return session;
}

/**
 * Loads the live session of the request's cookie: none without a cookie or
 * for an unknown id; an expired one is deleted; a live one is touched.
 */
export function loadSession(db: DatabaseSync, request: FastifyRequest, now: Date): Session | undefined {
  const id = request.cookies[COOKIE];
  if (!id) return undefined;
  const session = findSession(db, id);
  if (!session) return undefined;
  const lastSeen = new Date(session.last_seen_at).getTime();
  if (now.getTime() - lastSeen > SESSION_DAYS * DAY_MS) {
    deleteSession(db, id);
    return undefined;
  }
  touchSession(db, id, now);
  return { ...session, last_seen_at: now.toISOString() };
}

/** Ends the session of the request's cookie, if any, and clears the cookie. */
export function endSession(db: DatabaseSync, request: FastifyRequest, reply: FastifyReply): void {
  const id = request.cookies[COOKIE];
  if (id) deleteSession(db, id);
  reply.clearCookie(COOKIE, cookieOptions());
}
