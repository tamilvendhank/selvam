import express from "express";
import multer from "multer";
import os from "node:os";

import {
  getJobDetails,
  getJobsForList,
  refreshAllJobs,
  refreshJob,
  submitPromptBatchWithFiles,
  submitTemplatedPromptBatch
} from "../services/jobs-service.js";
import {
  createProcedureDefinition,
  getProcedureDetails,
  getProceduresForList,
  updateProcedureDefinition
} from "../services/procedures-service.js";
import {
  createExecutionDefinition,
  getProcedureExecutionDetails,
  getProcedureExecutionsForList,
  refreshProcedureExecutionById,
  startProcedureExecutionById
} from "../services/procedure-executions-service.js";

const router = express.Router();
const upload = multer({
  dest: os.tmpdir()
});

function getPromptsFromBody(body) {
  const rawPrompts = body?.prompts;

  if (Array.isArray(rawPrompts)) {
    return rawPrompts;
  }

  if (typeof rawPrompts === "string") {
    return [rawPrompts];
  }

  return [];
}

function getPromptEntriesFromBody(body) {
  const rawPayload = body?.submissionPayload ?? body?.["submissionPayload[]"];

  if (typeof rawPayload !== "undefined" && rawPayload !== null) {
    let parsedPayload = rawPayload;

    if (typeof rawPayload === "string") {
      parsedPayload = rawPayload.trim() ? JSON.parse(rawPayload) : [];
    }

    if (Array.isArray(parsedPayload)) {
      return parsedPayload.map((entry) => {
        if (typeof entry === "string") {
          return { query: entry };
        }

        return {
          query: typeof entry?.query === "string" ? entry.query : "",
          model: typeof entry?.model === "string" ? entry.model : "",
          reasoningEffort: typeof entry?.reasoningEffort === "string" ? entry.reasoningEffort : ""
        };
      });
    }
  }

  return getPromptsFromBody(body).map((query) => ({ query }));
}

router.get("/api/submissions", async (req, res, next) => {
  try {
    const jobs = await getJobsForList();
    res.json({ jobs });
  } catch (error) {
    next(error);
  }
});

router.post("/api/submissions", upload.any(), async (req, res, next) => {
  try {
    const promptEntries = getPromptEntriesFromBody(req.body);
    const filesByPromptIndex = new Map();

    for (const file of req.files || []) {
      const match = String(file.fieldname).match(/^promptFiles-(\d+)$/);

      if (!match) {
        continue;
      }

      const promptIndex = Number(match[1]);
      const existingFiles = filesByPromptIndex.get(promptIndex) || [];
      existingFiles.push(file);
      filesByPromptIndex.set(promptIndex, existingFiles);
    }

    const hasAtLeastOneSubmission = promptEntries.some((entry, index) => {
      const hasPrompt = typeof entry.query === "string" && entry.query.trim();
      const hasFiles = (filesByPromptIndex.get(index) || []).length > 0;
      return hasPrompt || hasFiles;
    });

    if (!hasAtLeastOneSubmission) {
      res.status(400).json({
        error: "Please add at least one prompt or file before submitting."
      });
      return;
    }

    const enrichedPromptEntries = promptEntries.map((entry, index) => ({
      query: entry.query,
      model: entry.model,
      reasoningEffort: entry.reasoningEffort,
      files: filesByPromptIndex.get(index) || []
    }));

    const jobs = await submitPromptBatchWithFiles(enrichedPromptEntries);
    res.status(201).json({ jobs });
  } catch (error) {
    next(error);
  }
});

router.post("/api/templated-submissions", async (req, res, next) => {
  try {
    const jobs = await submitTemplatedPromptBatch({
      promptTemplate: req.body?.promptTemplate,
      records: req.body?.records,
      model: req.body?.model,
      reasoningEffort: req.body?.reasoningEffort
    });

    res.status(201).json({ jobs });
  } catch (error) {
    next(error);
  }
});

router.get("/api/procedures", async (req, res, next) => {
  try {
    const procedures = await getProceduresForList();
    res.json({ procedures });
  } catch (error) {
    next(error);
  }
});

router.get("/api/procedures/:id", async (req, res, next) => {
  try {
    const procedure = await getProcedureDetails(req.params.id);

    if (!procedure) {
      res.status(404).json({ error: "Procedure not found." });
      return;
    }

    res.json({ procedure });
  } catch (error) {
    next(error);
  }
});

router.post("/api/procedures", async (req, res, next) => {
  try {
    const procedure = await createProcedureDefinition({
      name: req.body?.name,
      steps: req.body?.steps
    });

    res.status(201).json({ procedure });
  } catch (error) {
    next(error);
  }
});

router.put("/api/procedures/:id", async (req, res, next) => {
  try {
    const procedure = await updateProcedureDefinition(req.params.id, {
      name: req.body?.name,
      steps: req.body?.steps
    });

    if (!procedure) {
      res.status(404).json({ error: "Procedure not found." });
      return;
    }

    res.json({ procedure });
  } catch (error) {
    next(error);
  }
});

router.get("/api/procedure-executions", async (req, res, next) => {
  try {
    const executions = await getProcedureExecutionsForList();
    res.json({ executions });
  } catch (error) {
    next(error);
  }
});

router.get("/api/procedure-executions/:id", async (req, res, next) => {
  try {
    const execution = await getProcedureExecutionDetails(req.params.id);

    if (!execution) {
      res.status(404).json({ error: "Execution not found." });
      return;
    }

    res.json({ execution });
  } catch (error) {
    next(error);
  }
});

router.post("/api/procedure-executions", async (req, res, next) => {
  try {
    const execution = await createExecutionDefinition({
      procedureId: req.body?.procedureId,
      prompt: req.body?.prompt
    });

    res.status(201).json({ execution });
  } catch (error) {
    next(error);
  }
});

router.post("/api/procedure-executions/:id/start", async (req, res, next) => {
  try {
    const execution = await startProcedureExecutionById(req.params.id);

    if (!execution) {
      res.status(404).json({ error: "Execution not found." });
      return;
    }

    res.json({ execution });
  } catch (error) {
    next(error);
  }
});

router.post("/api/procedure-executions/:id/refresh", async (req, res, next) => {
  try {
    const execution = await refreshProcedureExecutionById(req.params.id);

    if (!execution) {
      res.status(404).json({ error: "Execution not found." });
      return;
    }

    res.json({ execution });
  } catch (error) {
    next(error);
  }
});

router.post("/api/submissions/refresh-all", async (req, res, next) => {
  try {
    const jobs = await refreshAllJobs();
    res.json({ jobs });
  } catch (error) {
    next(error);
  }
});

router.get("/api/submissions/:id", async (req, res, next) => {
  try {
    const job = await getJobDetails(req.params.id);

    if (!job) {
      res.status(404).json({ error: "Job not found." });
      return;
    }

    res.json({ job });
  } catch (error) {
    next(error);
  }
});

router.post("/api/submissions/:id/refresh", async (req, res, next) => {
  try {
    const job = await refreshJob(req.params.id);

    if (!job) {
      res.status(404).json({ error: "Job not found." });
      return;
    }

    res.json({ job });
  } catch (error) {
    next(error);
  }
});

router.get("/", (req, res) => {
  res.render("index");
});

router.get("/submissions", (req, res) => {
  res.render("index");
});

router.get("/submissions/:id", (req, res) => {
  res.render("index");
});

router.get("/templated-submissions", (req, res) => {
  res.render("index");
});

router.get("/procedures", (req, res) => {
  res.render("index");
});

router.get("/procedure-executions", (req, res) => {
  res.render("index");
});

router.get("/procedure-executions/:id", (req, res) => {
  res.render("index");
});

router.get("/platform", (req, res) => {
  res.render("index");
});

router.get("/platform/companies", (req, res) => {
  res.render("index");
});

router.get("/platform/companies/:id", (req, res) => {
  res.render("index");
});

router.get("/platform/reviews", (req, res) => {
  res.render("index");
});

router.get("/platform/reviews/:id", (req, res) => {
  res.render("index");
});

router.get("/platform/workflow-runs", (req, res) => {
  res.render("index");
});

router.get("/platform/workflow-runs/:id", (req, res) => {
  res.render("index");
});

router.get("/platform/ai-batch-jobs", (req, res) => {
  res.render("index");
});

router.get("/platform/ai-batch-jobs/:id", (req, res) => {
  res.render("index");
});

router.get("/platform/capital-allocations", (req, res) => {
  res.render("index");
});

router.get("/platform/capital-allocations/:id", (req, res) => {
  res.render("index");
});

router.get("/platform/positions", (req, res) => {
  res.render("index");
});

router.get("/platform/config", (req, res) => {
  res.render("index");
});

router.get("/platform/config/snapshots/:id", (req, res) => {
  res.render("index");
});

router.get("/platform/overrides", (req, res) => {
  res.render("index");
});

router.get("/jobs", (req, res) => {
  res.redirect(302, "/submissions");
});

router.get("/jobs/:id", (req, res) => {
  res.redirect(302, `/submissions/${req.params.id}`);
});

router.use((req, res, next) => {
  if (req.method === "GET" && !req.path.startsWith("/api/")) {
    res.status(404).render("index");
    return;
  }

  next();
});

export default router;
