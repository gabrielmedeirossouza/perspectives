import type { FastifyInstance } from "fastify";
import type { DatabaseSync } from "node:sqlite";
import { createAccount, findAccountByEmail, findAccountById } from "./db.js";
import { DECOY_HASH, hashPassword, verifyPassword } from "./password.js";
import { endSession, loadSession, startSession } from "./session.js";

export interface AuthOptions {
  db: DatabaseSync;
  now: () => Date;
}

interface Credentials {
  email?: unknown;
  password?: unknown;
}

export const MIN_PASSWORD_LENGTH = 8;

/** The /auth routes: sign-up, sign-in, sign-out, session. */
export async function authRoutes(app: FastifyInstance, options: AuthOptions): Promise<void> {
  const { db, now } = options;

  app.post<{ Body: Credentials }>("/auth/sign-up", async (request, reply) => {
    const { email, password } = request.body ?? {};
    if (typeof email !== "string" || !email.includes("@") || typeof password !== "string" || password.length < MIN_PASSWORD_LENGTH) {
      return reply.code(400).send({ error: "invalid" });
    }
    const account = createAccount(db, email, hashPassword(password), now());
    if (!account) return reply.code(409).send({ error: "email_taken" });
    startSession(db, reply, account.id, now());
    return reply.code(201).send({ id: account.id, email: account.email });
  });

  app.post<{ Body: Credentials }>("/auth/sign-in", async (request, reply) => {
    const { email, password } = request.body ?? {};
    const account = typeof email === "string" ? findAccountByEmail(db, email) : undefined;
    const matches = verifyPassword(typeof password === "string" ? password : "", account?.password_hash ?? DECOY_HASH);
    if (!account || !matches) return reply.code(401).send({ error: "invalid_credentials" });
    startSession(db, reply, account.id, now());
    return reply.code(200).send({ id: account.id, email: account.email });
  });

  app.post("/auth/sign-out", async (request, reply) => {
    endSession(db, request, reply);
    return reply.code(204).send();
  });

  app.get("/auth/session", async (request, reply) => {
    const session = loadSession(db, request, now());
    const account = session ? findAccountById(db, session.account_id) : undefined;
    if (!session || !account) return reply.code(401).send({ error: "no_session" });
    return reply.code(200).send({ id: account.id, email: account.email });
  });
}
