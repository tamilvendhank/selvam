import express from "express";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { config } from "./config.js";
import { closeDb, getDb } from "./db.js";
import jobsRouter from "./routes/jobs.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();

app.set("view engine", "ejs");
app.set("views", path.join(__dirname, "views"));

app.use(express.json());
app.use(express.urlencoded({ extended: false }));
app.use(express.static(path.join(__dirname, "public")));
app.use("/shared", express.static(path.join(__dirname, "shared")));
app.use("/vendor", express.static(path.join(__dirname, "..", "node_modules", "redom", "dist")));
app.use("/vendor-marked", express.static(path.join(__dirname, "..", "node_modules", "marked", "lib")));
app.use("/vendor-fontawesome", express.static(path.join(__dirname, "..", "node_modules", "@fortawesome", "fontawesome-free")));

app.locals.statusClassName = (status) => {
  if (status === "completed") {
    return "status-completed";
  }

  if (["failed", "expired", "cancelled", "submission_failed"].includes(status)) {
    return "status-failed";
  }

  return "status-active";
};

app.use("/", jobsRouter);

app.use((error, req, res, next) => {
  console.error(error);

  if (req.path.startsWith("/api/")) {
    res.status(500).json({
      error: error.message || "Something went wrong."
    });
    return;
  }

  res.status(500).render("index");
});

async function start() {
  await getDb();

  app.listen(config.port, () => {
    console.log(`Web app running at http://localhost:${config.port}`);
  });
}

start().catch((error) => {
  console.error("Failed to start application:", error);
  process.exit(1);
});

for (const signal of ["SIGINT", "SIGTERM"]) {
  process.on(signal, async () => {
    await closeDb();
    process.exit(0);
  });
}
