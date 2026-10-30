import Fastify, { type FastifyError, type FastifyInstance } from "fastify";
import cookie from "@fastify/cookie";
import type { DatabaseSync } from "node:sqlite";
import { authRoutes } from "./auth/routes.js";
import { migrate } from "./auth/db.js";

export interface BuildOptions {
  db: DatabaseSync;
}

/** Builds the configured API without listening; main.ts listens. */
export function build(options: BuildOptions): FastifyInstance {
  const app = Fastify();
  migrate(options.db);
  app.setNotFoundHandler((_request, reply) => reply.code(404).send({ error: "not_found" }));
  app.setErrorHandler((error: FastifyError, _request, reply) => {
    if (error.code?.startsWith("FST_ERR_CTP_")) return reply.code(400).send({ error: "invalid" });
    return reply.code(500).send({ error: "internal_error" });
  });
  app.register(cookie);
  app.register(authRoutes, { db: options.db });
  return app;
}
