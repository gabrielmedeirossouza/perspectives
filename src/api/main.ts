import { DatabaseSync } from "node:sqlite";
import { build } from "./server.js";

const db = new DatabaseSync(process.env.LEDGER_DB ?? "ledger.db");
const app = build({ db });
const port = Number(process.env.PORT ?? 3000);
app.listen({ port, host: "0.0.0.0" }).then((address) => {
  app.log.info(`listening on ${address}`);
});
