import {
  createProcedureExecution,
  getProcedureExecutionById,
  listProcedureExecutions,
  updateProcedureExecution
} from "../repositories/procedure-executions-repository.js";
import { getProcedureById } from "../repositories/procedures-repository.js";
import { refreshJob, submitPromptBatchWithFiles } from "./jobs-service.js";

function buildExecutionPrompt(step, executionPrompt, previousResultText) {
  const parts = [step.prompt];
  const trimmedExecutionPrompt = typeof executionPrompt === "string" ? executionPrompt.trim() : "";
  const trimmedPreviousResult = typeof previousResultText === "string" ? previousResultText.trim() : "";

  if (trimmedExecutionPrompt) {
    parts.push(`Initial input:\n${trimmedExecutionPrompt}`);
  }

  if (trimmedPreviousResult) {
    parts.push(`Previous step output:\n${trimmedPreviousResult}`);
  }

  return parts.join("\n\n");
}

function createExecutionStepSnapshot(step, index) {
  return {
    id: step.id || `step-${index + 1}`,
    stepNumber: index + 1,
    prompt: step.prompt,
    model: step.model,
    reasoningEffort: step.reasoningEffort || null,
    status: "pending",
    stepInput: "",
    jobId: null,
    batchId: null,
    startedAt: null,
    completedAt: null,
    lastSyncedAt: null,
    executionDurationMs: null,
    resultText: "",
    resultResponseBody: null,
    latestError: null
  };
}

function toExecutionStepViewModel(step) {
  const startedAt = step.startedAt ? new Date(step.startedAt) : null;
  const completedAt = step.completedAt ? new Date(step.completedAt) : null;
  const durationMs =
    typeof step.executionDurationMs === "number"
      ? step.executionDurationMs
      : startedAt && completedAt
        ? completedAt.getTime() - startedAt.getTime()
        : null;

  return {
    ...step,
    canRefresh: step.status === "in_progress" && Boolean(step.jobId),
    createdInputLabel: step.stepInput || "N/A",
    startedAtLabel: startedAt ? startedAt.toLocaleString() : "Not started",
    completedAtLabel: completedAt ? completedAt.toLocaleString() : null,
    durationLabel: durationMs !== null ? `${Math.max(0, Math.round(durationMs / 1000))}s` : "N/A"
  };
}

function toViewModel(execution) {
  if (!execution) {
    return null;
  }

  return {
    ...execution,
    steps: Array.isArray(execution.steps) ? execution.steps.map(toExecutionStepViewModel) : [],
    createdAtLabel: execution.createdAt ? new Date(execution.createdAt).toLocaleString() : "Unknown",
    updatedAtLabel: execution.updatedAt ? new Date(execution.updatedAt).toLocaleString() : "Unknown",
    startedAtLabel: execution.startedAt ? new Date(execution.startedAt).toLocaleString() : null,
    completedAtLabel: execution.completedAt ? new Date(execution.completedAt).toLocaleString() : null,
    initialPromptLabel:
      typeof execution.initialPrompt === "string" && execution.initialPrompt.trim() ? execution.initialPrompt.trim() : "N/A",
    currentStepNumber:
      typeof execution.currentStepIndex === "number" && execution.currentStepIndex >= 0
        ? execution.currentStepIndex + 1
        : null
  };
}

function getCurrentStepIndex(execution) {
  return execution.steps.findIndex((step) => step.status === "in_progress");
}

function getNextPendingStepIndex(execution) {
  return execution.steps.findIndex((step) => step.status === "pending");
}

async function persistExecution(executionId, execution, updates = {}) {
  return updateProcedureExecution(executionId, {
    ...updates,
    status: execution.status,
    currentStepIndex: execution.currentStepIndex,
    startedAt: execution.startedAt,
    completedAt: execution.completedAt,
    lastRefreshedAt: execution.lastRefreshedAt,
    steps: execution.steps
  });
}

async function startExecutionStep(execution, stepIndex) {
  const step = execution.steps[stepIndex];

  if (!step || step.status !== "pending") {
    return execution;
  }

  const previousStep = stepIndex > 0 ? execution.steps[stepIndex - 1] : null;
  const stepInput = buildExecutionPrompt(step, execution.initialPrompt, previousStep?.resultText || "");
  const [job] = await submitPromptBatchWithFiles(
    [
      {
        query: stepInput,
        model: step.model,
        reasoningEffort: step.reasoningEffort
      }
    ],
    { submissionType: "procedure_execution" }
  );

  const now = new Date();
  execution.status = "running";
  execution.startedAt = execution.startedAt || now;
  execution.currentStepIndex = stepIndex;
  execution.lastRefreshedAt = now;
  execution.steps[stepIndex] = {
    ...step,
    status: "in_progress",
    stepInput,
    jobId: job.id,
    batchId: job.batchId || null,
    startedAt: now,
    lastSyncedAt: job.lastSyncedAt || now,
    resultText: job.resultText || "",
    resultResponseBody: job.resultResponseBody || null,
    latestError: job.latestErrorLine?.error || null
  };

  return execution;
}

async function refreshActiveExecutionStep(execution) {
  const activeStepIndex = getCurrentStepIndex(execution);

  if (activeStepIndex === -1) {
    return execution;
  }

  const activeStep = execution.steps[activeStepIndex];
  const job = await refreshJob(activeStep.jobId);
  const startedAt = activeStep.startedAt ? new Date(activeStep.startedAt) : null;
  const completedAt = job?.completedAt ? new Date(job.completedAt) : null;

  execution.lastRefreshedAt = new Date();
  execution.steps[activeStepIndex] = {
    ...activeStep,
    batchId: job?.batchId || activeStep.batchId || null,
    lastSyncedAt: job?.lastSyncedAt || execution.lastRefreshedAt,
    resultText: job?.resultText || "",
    resultResponseBody: job?.resultResponseBody || null,
    latestError: job?.latestErrorLine?.error || null
  };

  if (!job) {
    execution.status = "failed";
    execution.currentStepIndex = activeStepIndex;
    execution.steps[activeStepIndex].status = "failed";
    execution.steps[activeStepIndex].latestError = {
      message: "Linked batch job could not be found."
    };
    return execution;
  }

  if (job.status === "completed") {
    execution.steps[activeStepIndex] = {
      ...execution.steps[activeStepIndex],
      status: "completed",
      completedAt,
      executionDurationMs: startedAt && completedAt ? completedAt.getTime() - startedAt.getTime() : null
    };
    execution.currentStepIndex = null;
  } else if (["failed", "expired", "cancelled", "submission_failed"].includes(job.status)) {
    execution.status = "failed";
    execution.currentStepIndex = activeStepIndex;
    execution.steps[activeStepIndex] = {
      ...execution.steps[activeStepIndex],
      status: "failed",
      completedAt: completedAt || new Date(),
      executionDurationMs: startedAt ? Date.now() - startedAt.getTime() : null
    };
  }

  return execution;
}

async function progressExecution(execution) {
  await refreshActiveExecutionStep(execution);

  if (execution.status === "failed") {
    return execution;
  }

  const nextPendingStepIndex = getNextPendingStepIndex(execution);

  if (nextPendingStepIndex === -1 && getCurrentStepIndex(execution) === -1) {
    execution.status = "completed";
    execution.completedAt = execution.completedAt || new Date();
    execution.currentStepIndex = null;
    return execution;
  }

  if (getCurrentStepIndex(execution) === -1 && nextPendingStepIndex !== -1) {
    await startExecutionStep(execution, nextPendingStepIndex);
  }

  return execution;
}

export async function getProcedureExecutionsForList() {
  const executions = await listProcedureExecutions();
  return executions.map(toViewModel);
}

export async function getProcedureExecutionDetails(id) {
  const execution = await getProcedureExecutionById(id);
  return toViewModel(execution);
}

export async function createExecutionDefinition({ procedureId, prompt }) {
  const procedure = await getProcedureById(procedureId);

  if (!procedure) {
    throw new Error("Procedure not found.");
  }

  const now = new Date();
  const execution = await createProcedureExecution({
    procedureId: procedure.id,
    procedureName: procedure.name,
    initialPrompt: typeof prompt === "string" ? prompt.trim() : "",
    status: "draft",
    currentStepIndex: null,
    startedAt: null,
    completedAt: null,
    lastRefreshedAt: null,
    steps: (Array.isArray(procedure.steps) ? procedure.steps : []).map(createExecutionStepSnapshot),
    createdAt: now,
    updatedAt: now
  });

  return toViewModel(execution);
}

export async function startProcedureExecutionById(id) {
  const execution = await getProcedureExecutionById(id);

  if (!execution) {
    return null;
  }

  if (execution.status === "completed") {
    return toViewModel(execution);
  }

  if (getCurrentStepIndex(execution) === -1) {
    const nextPendingStepIndex = getNextPendingStepIndex(execution);

    if (nextPendingStepIndex !== -1) {
      await startExecutionStep(execution, nextPendingStepIndex);
    }
  }

  const updatedExecution = await persistExecution(id, execution);
  return toViewModel(updatedExecution);
}

export async function refreshProcedureExecutionById(id) {
  const execution = await getProcedureExecutionById(id);

  if (!execution) {
    return null;
  }

  if (execution.status === "draft" || execution.status === "completed" || execution.status === "failed") {
    return toViewModel(execution);
  }

  await progressExecution(execution);
  const updatedExecution = await persistExecution(id, execution);
  return toViewModel(updatedExecution);
}
