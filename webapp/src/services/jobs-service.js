import { config } from "../config.js";
import {
  createJobs,
  getJobById,
  listJobs,
  listJobsByBatchId,
  listJobsByStatuses,
  updateJob
} from "../repositories/jobs-repository.js";
import {
  buildBatchRequestBody,
  buildBatchSnapshot,
  deleteOpenAiInputFiles,
  loadBatchFileJsonl,
  retrieveOpenAiBatch,
  submitOpenAiBatch,
  uploadOpenAiInputFiles
} from "./openai-batch-service.js";
import { normalizeModelName, normalizeReasoningEffort } from "../shared/openai-models.js";

const ACTIVE_BATCH_STATUSES = new Set(["validating", "in_progress", "finalizing", "cancelling"]);
const TERMINAL_BATCH_STATUSES = new Set(["completed", "failed", "expired", "cancelled"]);

function toDateFromUnixSeconds(value) {
  if (!value) {
    return null;
  }

  return new Date(value * 1000);
}

function normalizePromptInputs(rawPrompts) {
  const prompts = Array.isArray(rawPrompts) ? rawPrompts : [rawPrompts];

  return prompts.map((prompt) => (typeof prompt === "string" ? prompt.trim() : "")).filter(Boolean);
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function resolveTemplateValue(record, path) {
  return path.split(".").reduce((value, key) => {
    if (value === null || typeof value === "undefined") {
      return undefined;
    }

    return value[key];
  }, record);
}

function renderPromptTemplate(template, record, index) {
  return template.replace(/\{([^{}]+)\}/g, (match, token) => {
    const variableName = token.trim();
    const value = resolveTemplateValue(record, variableName);

    if (typeof value === "undefined") {
      throw new Error(`Missing template variable "${variableName}" for item ${index + 1}.`);
    }

    return String(value);
  });
}

function normalizeTemplateRecords(rawRecords) {
  if (!Array.isArray(rawRecords)) {
    throw new Error("JSON array input must be an array.");
  }

  return rawRecords.map((record, index) => {
    if (!record || typeof record !== "object" || Array.isArray(record)) {
      throw new Error(`JSON array item ${index + 1} must be an object.`);
    }

    return record;
  });
}

function getOutputLineForJob(job, outputLines) {
  return outputLines.find((line) => line.custom_id === job.customId) || null;
}

function getErrorLineForJob(job, errorLines) {
  return errorLines.find((line) => line.custom_id === job.customId) || null;
}

function extractResponseText(responseBody) {
  if (!responseBody) {
    return "";
  }

  if (typeof responseBody.output_text === "string" && responseBody.output_text.trim()) {
    return responseBody.output_text;
  }

  if (!Array.isArray(responseBody.output)) {
    return "";
  }

  const textParts = [];

  for (const outputItem of responseBody.output) {
    if (!Array.isArray(outputItem?.content)) {
      continue;
    }

    for (const contentItem of outputItem.content) {
      if (contentItem?.type === "output_text" && typeof contentItem.text === "string" && contentItem.text) {
        textParts.push(contentItem.text);
      }
    }
  }

  return textParts.join("\n\n");
}

function buildResultSnapshot(job, outputLines, errorLines) {
  const outputLine = getOutputLineForJob(job, outputLines);
  const errorLine = getErrorLineForJob(job, errorLines);
  const responseBody = outputLine?.response?.body || null;
  const extractedText = extractResponseText(responseBody);

  return {
    resultText: extractedText,
    resultResponseBody: responseBody,
    latestOutputLine: outputLine,
    latestErrorLine: errorLine
  };
}

function toViewModel(job) {
  if (!job) {
    return null;
  }

  const isCompleted = job.status === "completed";
  const isActive = ACTIVE_BATCH_STATUSES.has(job.status);
  const isTerminal = TERMINAL_BATCH_STATUSES.has(job.status);
  const resolvedResultText = job.resultText || extractResponseText(job.resultResponseBody);
  const normalizedQuery = typeof job.query === "string" ? job.query.trim() : "";
  const model = normalizeModelName(job.model || job.resultResponseBody?.model || config.openai.model);
  const reasoningEffort = normalizeReasoningEffort(model, job.reasoningEffort);

  return {
    ...job,
    model,
    query: normalizedQuery,
    queryLabel: normalizedQuery || "N/A",
    reasoningEffort,
    reasoningEffortLabel: reasoningEffort || "N/A",
    resultText: resolvedResultText,
    createdAtLabel: job.createdAt ? new Date(job.createdAt).toLocaleString() : "Unknown",
    updatedAtLabel: job.updatedAt ? new Date(job.updatedAt).toLocaleString() : "Unknown",
    lastSyncedAtLabel: job.lastSyncedAt ? new Date(job.lastSyncedAt).toLocaleString() : "Never",
    completedAtLabel: job.completedAt ? new Date(job.completedAt).toLocaleString() : null,
    previewText: resolvedResultText ? resolvedResultText.slice(0, 180) : "",
    canRefresh: isActive,
    canViewResults: isCompleted && Boolean(resolvedResultText),
    isCompleted,
    isActive,
    isTerminal
  };
}

function buildBatchInputLine(job) {
  return {
    custom_id: job.customId,
    method: "POST",
    url: config.openai.batchEndpoint,
    body: buildBatchRequestBody(job.query, job.attachedFiles || [], job.model, job.reasoningEffort)
  };
}

function buildSubmissionJobs(promptEntries, metadata = {}) {
  const now = new Date();
  const submissionId = `submission-${now.getTime()}`;

  return promptEntries.map((entry, index) => ({
    query: entry.query,
    customId: "",
    submissionId,
    submissionIndex: index + 1,
    submissionSize: promptEntries.length,
    submissionType: metadata.submissionType || "manual",
    promptTemplate: metadata.promptTemplate || null,
    templateRecord: entry.templateRecord || null,
    attachedFiles: entry.attachedFiles || [],
    model: normalizeModelName(entry.model || config.openai.model),
    reasoningEffort: normalizeReasoningEffort(entry.model || config.openai.model, entry.reasoningEffort),
    status: "preparing",
    batchId: null,
    inputFileId: null,
    outputFileId: null,
    errorFileId: null,
    requestCounts: null,
    resultText: "",
    resultResponseBody: null,
    latestOutputLine: null,
    latestErrorLine: null,
    lastSyncedAt: null,
    completedAt: null,
    openaiBatch: null,
    createdAt: now,
    updatedAt: now
  }));
}

async function markJobsAsSubmissionFailed(jobs, error) {
  await Promise.all(
    jobs.map((job) =>
      updateJob(job.id, {
        status: "submission_failed",
        latestErrorLine: {
          error: {
            message: error.message
          }
        }
      })
    )
  );
}

async function refreshSharedBatchJobs(batchId, preferredJobId) {
  const jobs = await listJobsByBatchId(batchId);

  if (!jobs.length) {
    return null;
  }

  const batch = await retrieveOpenAiBatch(batchId);
  const outputLines = batch.output_file_id ? await loadBatchFileJsonl(batch.output_file_id) : [];
  const errorLines = batch.error_file_id ? await loadBatchFileJsonl(batch.error_file_id) : [];

  const updates = jobs.map((job) => {
    const resultSnapshot = buildResultSnapshot(job, outputLines, errorLines);

    return updateJob(job.id, {
      status: batch.status,
      outputFileId: batch.output_file_id || null,
      errorFileId: batch.error_file_id || null,
      requestCounts: batch.request_counts || null,
      lastSyncedAt: new Date(),
      completedAt: toDateFromUnixSeconds(batch.completed_at),
      openaiBatch: buildBatchSnapshot(batch),
      ...resultSnapshot
    });
  });

  const refreshedJobs = await Promise.all(updates);
  const selectedJob = refreshedJobs.find((job) => job.id === preferredJobId) || refreshedJobs[0];

  return toViewModel(selectedJob);
}

export async function getJobsForList() {
  const jobs = await listJobs();
  return jobs.map(toViewModel);
}

export async function getJobDetails(id) {
  const job = await getJobById(id);
  return toViewModel(job);
}

async function submitPromptEntries(promptEntries, metadata = {}) {
  if (!promptEntries.length) {
    throw new Error("At least one prompt is required.");
  }

  const initialJobs = await createJobs(buildSubmissionJobs(promptEntries, metadata));
  const jobsWithCustomIds = await Promise.all(
    initialJobs.map((job) => {
      const customId = `job-${job.id}`;

      return updateJob(job.id, { customId });
    })
  );

  try {
    const { inputFile, batch } = await submitOpenAiBatch({
      fileStem: jobsWithCustomIds[0].submissionId,
      lines: jobsWithCustomIds.map((job) => buildBatchInputLine(job)),
      metadata: {
        app: "webapp",
        submission_id: jobsWithCustomIds[0].submissionId,
        job_count: String(jobsWithCustomIds.length),
        submission_type: metadata.submissionType || "manual"
      }
    });

    const updatedJobs = await Promise.all(
      jobsWithCustomIds.map((job) =>
        updateJob(job.id, {
          status: batch.status,
          batchId: batch.id,
          inputFileId: inputFile.id,
          outputFileId: batch.output_file_id || null,
          errorFileId: batch.error_file_id || null,
          requestCounts: batch.request_counts || null,
          lastSyncedAt: new Date(),
          openaiBatch: buildBatchSnapshot(batch)
        })
      )
    );

    return updatedJobs.map(toViewModel);
  } catch (error) {
    await markJobsAsSubmissionFailed(jobsWithCustomIds, error);
    throw error;
  }
}

export async function submitPromptBatch(rawPrompts, metadata = {}) {
  const prompts = normalizePromptInputs(rawPrompts);

  return submitPromptEntries(
    prompts.map((query) => ({ query, attachedFiles: [] })),
    { submissionType: "manual", ...metadata }
  );
}

export async function submitPromptBatchWithFiles(promptEntries, metadata = {}) {
  if (!Array.isArray(promptEntries) || !promptEntries.length) {
    throw new Error("At least one prompt is required.");
  }

  const normalizedEntries = [];
  const allUploadedFiles = [];

  try {
    for (const entry of promptEntries) {
      const query = typeof entry?.query === "string" ? entry.query.trim() : "";
      const files = Array.isArray(entry?.files) ? entry.files : [];
      const model = normalizeModelName(entry?.model || config.openai.model);
      const reasoningEffort = normalizeReasoningEffort(model, entry?.reasoningEffort);

      if (!query && !files.length) {
        continue;
      }

      const uploadedFiles = await uploadOpenAiInputFiles(files);
      allUploadedFiles.push(...uploadedFiles);

      normalizedEntries.push({
        query,
        attachedFiles: uploadedFiles,
        model,
        reasoningEffort
      });
    }

    if (!normalizedEntries.length) {
      throw new Error("At least one prompt or file is required.");
    }

    return await submitPromptEntries(normalizedEntries, { submissionType: "manual", ...metadata });
  } catch (error) {
    await deleteOpenAiInputFiles(allUploadedFiles);
    throw error;
  }
}

export async function submitTemplatedPromptBatch({ promptTemplate, records, model, reasoningEffort }) {
  const trimmedTemplate = typeof promptTemplate === "string" ? promptTemplate.trim() : "";

  if (!trimmedTemplate) {
    throw new Error("Prompt template is required.");
  }

  const normalizedRecords = normalizeTemplateRecords(records);
  const resolvedModel = normalizeModelName(model || config.openai.model);
  const resolvedReasoningEffort = normalizeReasoningEffort(resolvedModel, reasoningEffort);
  const promptEntries = normalizedRecords.map((record, index) => ({
    query: renderPromptTemplate(trimmedTemplate, record, index),
    templateRecord: record,
    model: resolvedModel,
    reasoningEffort: resolvedReasoningEffort
  }));

  return submitPromptEntries(promptEntries, {
    submissionType: "templated",
    promptTemplate: trimmedTemplate
  });
}

export async function refreshJob(id) {
  const job = await getJobById(id);

  if (!job) {
    return null;
  }

  if (!job.batchId) {
    return toViewModel(job);
  }

  return refreshSharedBatchJobs(job.batchId, id);
}

export async function refreshAllJobs() {
  const activeJobs = await listJobsByStatuses(Array.from(ACTIVE_BATCH_STATUSES));
  const batchEntries = new Map();

  for (const job of activeJobs) {
    if (job.batchId && !batchEntries.has(job.batchId)) {
      batchEntries.set(job.batchId, job.id);
    }
  }

  await Promise.all(
    Array.from(batchEntries.entries()).map(([batchId, preferredJobId]) =>
      refreshSharedBatchJobs(batchId, preferredJobId)
    )
  );

  return getJobsForList();
}
