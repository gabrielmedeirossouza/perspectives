import Fastify, { type FastifyInstance } from "fastify";
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
  app.register(cookie);
  app.register(authRoutes, { db: options.db });
  return app;
}
