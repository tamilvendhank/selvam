import fs from "node:fs";
import { promises as fsPromises } from "node:fs";
import { promises as fsp } from "node:fs";
import os from "node:os";
import path from "node:path";

import { config } from "../config.js";
import { normalizeModelName, normalizeReasoningEffort } from "../shared/openai-models.js";
import { openai } from "./openai-client.js";

function parseJsonl(text) {
  if (!text) {
    return [];
  }

  return text
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => JSON.parse(line));
}

async function createTempBatchFile(fileStem, lines) {
  const tempDir = await fsp.mkdtemp(path.join(os.tmpdir(), "openai-batch-webapp-"));
  const tempFilePath = path.join(tempDir, `${fileStem}.jsonl`);
  const jsonl = `${lines.map((line) => JSON.stringify(line)).join("\n")}\n`;

  await fsp.writeFile(tempFilePath, jsonl, "utf8");

  return {
    tempDir,
    tempFilePath
  };
}

async function removeTempBatchFile(tempDir) {
  if (!tempDir) {
    return;
  }

  await fsp.rm(tempDir, { recursive: true, force: true });
}

function buildMessageInput(query, attachedFiles = []) {
  const content = [];

  if (query) {
    content.push({
      type: "input_text",
      text: query
    });
  }

  for (const attachedFile of attachedFiles) {
    content.push({
      type: "input_file",
      file_id: attachedFile.openAiFileId
    });
  }

  return [
    {
      role: "user",
      content
    }
  ];
}

export function buildBatchRequestBody(query, attachedFiles = [], model, reasoningEffort) {
  const resolvedModel = normalizeModelName(model || config.openai.model);
  const resolvedReasoningEffort = normalizeReasoningEffort(resolvedModel, reasoningEffort);
  const requestBody = {
    model: resolvedModel,
    instructions: config.openai.responseInstructions,
    input: buildMessageInput(query, attachedFiles),
    text: {
      format: {
        type: "text"
      }
    }
  };

  if (resolvedReasoningEffort) {
    requestBody.reasoning = {
      effort: resolvedReasoningEffort
    };
  }

  return requestBody;
}

export function buildBatchSnapshot(batch) {
  return {
    id: batch.id,
    status: batch.status,
    endpoint: batch.endpoint,
    inputFileId: batch.input_file_id,
    outputFileId: batch.output_file_id || null,
    errorFileId: batch.error_file_id || null,
    requestCounts: batch.request_counts || null,
    errors: batch.errors || null,
    createdAt: batch.created_at || null,
    completedAt: batch.completed_at || null,
    failedAt: batch.failed_at || null,
    cancelledAt: batch.cancelled_at || null,
    expiredAt: batch.expired_at || null
  };
}

export async function submitOpenAiBatch({ fileStem, lines, metadata }) {
  let tempDir;

  try {
    const tempFiles = await createTempBatchFile(fileStem, lines);
    tempDir = tempFiles.tempDir;

    const inputFile = await openai.files.create({
      file: fs.createReadStream(tempFiles.tempFilePath),
      purpose: "batch",
      expires_after: {
        anchor: "created_at",
        seconds: 60 * 60 * 24 * 30
      }
    });

    const batch = await openai.batches.create({
      input_file_id: inputFile.id,
      endpoint: config.openai.batchEndpoint,
      completion_window: config.openai.completionWindow,
      metadata
    });

    return {
      inputFile,
      batch
    };
  } finally {
    await removeTempBatchFile(tempDir);
  }
}

export async function retrieveOpenAiBatch(batchId) {
  return openai.batches.retrieve(batchId);
}

export async function loadBatchFileJsonl(fileId) {
  if (!fileId) {
    return [];
  }

  const response = await openai.files.content(fileId);
  const text = await response.text();

  return parseJsonl(text);
}

export async function uploadOpenAiInputFiles(files) {
  const uploadedFiles = [];

  try {
    for (const file of files) {
      const uploadedFile = await openai.files.create({
        file: fs.createReadStream(file.path),
        purpose: "user_data"
      });

      uploadedFiles.push({
        openAiFileId: uploadedFile.id,
        originalName: file.originalname,
        mimeType: file.mimetype || "application/octet-stream",
        size: file.size || null
      });
    }

    await Promise.all(uploadedFiles.map((file) => openai.files.waitForProcessing(file.openAiFileId)));
    return uploadedFiles;
  } finally {
    await Promise.all(
      files.map(async (file) => {
        if (file?.path) {
          await fsPromises.rm(file.path, { force: true });
        }
      })
    );
  }
}

export async function deleteOpenAiInputFiles(files) {
  await Promise.all(
    files.map(async (file) => {
      if (file?.openAiFileId) {
        try {
          await openai.files.delete(file.openAiFileId);
        } catch (error) {
          // Ignore cleanup failures so the primary error can surface.
        }
      }
    })
  );
}
