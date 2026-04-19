#!/usr/bin/env node

import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
import OpenAI from "openai";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const STATE_FILE = path.join(__dirname, "openai-batch-state.json");
const CONFIG_FILE = path.join(__dirname, "batch-config.json");
const DEFAULT_COMPANIES_JSON = path.join(__dirname, "companies.json");
const DEFAULT_COMPANIES_TXT = path.join(__dirname, "companies.txt");

const DEFAULT_PROMPT =
  "Assess this company on business quality, management quality, financial strength, growth runway, valuation, risks, and potential mispricing.";

const REASONING_EFFORTS = new Set([
  "none",
  "minimal",
  "low",
  "medium",
  "high",
  "xhigh",
]);

const REASONING_SUMMARIES = new Set(["auto", "concise", "detailed"]);

function nowIso() {
  return new Date().toISOString();
}

function nowStamp() {
  return nowIso().replace(/[:.]/g, "-");
}

function exists(filePath) {
  try {
    fs.accessSync(filePath, fs.constants.F_OK);
    return true;
  } catch {
    return false;
  }
}

function readJson(filePath, fallback = null) {
  try {
    return JSON.parse(fs.readFileSync(filePath, "utf8"));
  } catch {
    return fallback;
  }
}

function writeJson(filePath, value) {
  fs.writeFileSync(filePath, JSON.stringify(value, null, 2) + "\n", "utf8");
}

function normalizeCompanyName(name) {
  return String(name ?? "").trim().replace(/\s+/g, " ");
}

function companyKey(name) {
  return normalizeCompanyName(name).toLowerCase();
}

function slug(value) {
  return String(value)
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 80) || "item";
}

function uniqueCompanies(items) {
  const seen = new Set();
  const result = [];

  for (const item of items || []) {
    const company = normalizeCompanyName(item);
    if (!company) continue;

    const key = companyKey(company);
    if (seen.has(key)) continue;

    seen.add(key);
    result.push(company);
  }

  return result;
}

function parseArgs(argv) {
  const args = {
    submit: false,
    statusOnly: false,
    input: null,
    model: null,
    prompt: null,
    maxOutputTokens: null,
    reasoningEffort: null,
    reasoningSummary: null,
    config: null,
    help: false,
  };

  for (const arg of argv) {
    if (arg === "--submit") args.submit = true;
    else if (arg === "--status" || arg === "--status-only") args.statusOnly = true;
    else if (arg === "--help" || arg === "-h") args.help = true;
    else if (arg.startsWith("--input=")) args.input = arg.slice("--input=".length);
    else if (arg.startsWith("--model=")) args.model = arg.slice("--model=".length);
    else if (arg.startsWith("--prompt=")) args.prompt = arg.slice("--prompt=".length);
    else if (arg.startsWith("--max-output-tokens=")) {
      args.maxOutputTokens = Number(arg.slice("--max-output-tokens=".length));
    } else if (arg.startsWith("--thinking-effort=") || arg.startsWith("--reasoning-effort=")) {
      args.reasoningEffort = arg.split("=")[1];
    } else if (arg.startsWith("--thinking-summary=") || arg.startsWith("--reasoning-summary=")) {
      args.reasoningSummary = arg.split("=")[1];
    } else if (arg.startsWith("--config=")) {
      args.config = arg.slice("--config=".length);
    }
  }

  if (args.statusOnly) args.submit = false;
  return args;
}

function printHelp() {
  console.log(`
Usage:
  node batch-company-assessments.mjs --submit
  node batch-company-assessments.mjs --status
  node batch-company-assessments.mjs --submit --input=companies.json
  node batch-company-assessments.mjs --submit --model=gpt-5.1
  node batch-company-assessments.mjs --submit --thinking-effort=low --thinking-summary=auto

Files in same directory:
  batch-config.json     optional config
  companies.json        optional array input
  companies.txt         optional line-based input
  openai-batch-state.json  persisted state

Behavior:
  - rerun-safe
  - prints status of previous batches every run
  - only submits NEW companies not seen before
  - downloads completed outputs/errors
  - writes parsed result JSON files

Supported company input formats:
  companies.json -> ["AbbVie Inc", "Exxon Mobil Corp"]
  companies.json -> { "companies": ["AbbVie Inc", "Exxon Mobil Corp"] }
  companies.txt  -> one company per line
`);
}

function loadConfig(configPathArg) {
  const configPath = configPathArg ?
    path.resolve(process.cwd(), configPathArg) :
    CONFIG_FILE;

  return exists(configPath) ? readJson(configPath, {}) : {};
}

function resolveConfig(args, fileConfig) {
  const envModel = process.env.OPENAI_MODEL || null;

  const config = {
    model: args.model ?? fileConfig.model ?? envModel ?? "gpt-5.1",
    prompt: args.prompt ?? fileConfig.prompt ?? DEFAULT_PROMPT,
    inputFile: args.input ?? fileConfig.inputFile ?? null,
    maxOutputTokens: args.maxOutputTokens ??
      fileConfig.maxOutputTokens ??
      null,
    reasoning: {
      effort: args.reasoningEffort ??
        fileConfig?.reasoning?.effort ??
        null,
      summary: args.reasoningSummary ??
        fileConfig?.reasoning?.summary ??
        null,
    },
  };

  validateConfig(config);
  return config;
}

function validateConfig(config) {
  if (!config.model) {
    throw new Error("Model is required.");
  }

  if (config.maxOutputTokens != null) {
    if (!Number.isFinite(config.maxOutputTokens) || config.maxOutputTokens <= 0) {
      throw new Error("maxOutputTokens must be a positive number.");
    }
  }

  if (config.reasoning.effort && !REASONING_EFFORTS.has(config.reasoning.effort)) {
    throw new Error(`Invalid reasoning effort: ${config.reasoning.effort}`);
  }

  if (config.reasoning.summary && !REASONING_SUMMARIES.has(config.reasoning.summary)) {
    throw new Error(`Invalid reasoning summary: ${config.reasoning.summary}`);
  }
}

function loadCompanies(inputFileFromConfig) {
  const inputPath = inputFileFromConfig ?
    path.resolve(process.cwd(), inputFileFromConfig) :
    exists(DEFAULT_COMPANIES_JSON) ?
    DEFAULT_COMPANIES_JSON :
    exists(DEFAULT_COMPANIES_TXT) ?
    DEFAULT_COMPANIES_TXT :
    null;

  if (!inputPath) {
    throw new Error(
      "No company input file found. Add companies.json or companies.txt in the script directory, or pass --input=..."
    );
  }

  const ext = path.extname(inputPath).toLowerCase();

  if (ext === ".json") {
    const data = readJson(inputPath);
    const items = Array.isArray(data) ? data : data?.companies;
    if (!Array.isArray(items)) {
      throw new Error("JSON input must be an array or { companies: [...] }.");
    }

    return {
      source: inputPath,
      companies: uniqueCompanies(items),
    };
  }

  if (ext === ".txt" || ext === ".csv") {
    const items = fs
      .readFileSync(inputPath, "utf8")
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter(Boolean)
      .map((line) => line.replace(/^[-*]\s*/, ""));

    return {
      source: inputPath,
      companies: uniqueCompanies(items),
    };
  }

  throw new Error(`Unsupported input file type: ${inputPath}`);
}

function loadState() {
  return readJson(STATE_FILE, {
    batches: [],
    submittedCompanies: {},
  });
}

function saveState(state) {
  writeJson(STATE_FILE, state);
}

function upsertBatch(state, batchRecord) {
  const index = state.batches.findIndex((x) => x.batchId === batchRecord.batchId);
  if (index >= 0) state.batches[index] = { ...state.batches[index], ...batchRecord };
  else state.batches.unshift(batchRecord);
}

function registerSubmittedCompanies(state, companies, batchId) {
  for (const company of companies) {
    const key = companyKey(company);
    if (!state.submittedCompanies[key]) {
      state.submittedCompanies[key] = {
        company,
        firstBatchId: batchId,
        submittedAt: nowIso(),
      };
    }
  }
}

function getNewCompanies(state, companies) {
  return companies.filter((company) => !state.submittedCompanies[companyKey(company)]);
}

function buildReasoning(config) {
  const reasoning = {};
  if (config.reasoning.effort) reasoning.effort = config.reasoning.effort;
  if (config.reasoning.summary) reasoning.summary = config.reasoning.summary;
  return Object.keys(reasoning).length ? reasoning : undefined;
}

function buildResponseBody(company, config) {
  const body = {
    model: config.model,
    input: `Company: ${company}\n\n${config.prompt}`,
    text: {
      format: {
        type: "text",
      },
    },
  };

  const reasoning = buildReasoning(config);
  if (reasoning) body.reasoning = reasoning;
  if (config.maxOutputTokens != null) body.max_output_tokens = config.maxOutputTokens;

  return body;
}

function buildJsonlLine(company, index, config) {
  return {
    custom_id: `${String(index + 1).padStart(4, "0")}-${slug(company)}`,
    method: "POST",
    url: "/v1/responses",
    body: buildResponseBody(company, config),
  };
}

function writeBatchInputFile(companies, config) {
  const fileName = `batch-input-${nowStamp()}.jsonl`;
  const filePath = path.join(__dirname, fileName);

  const lines = companies
    .map((company, index) => JSON.stringify(buildJsonlLine(company, index, config)))
    .join("\n") + "\n";

  fs.writeFileSync(filePath, lines, "utf8");

  return { fileName, filePath };
}

async function uploadBatchInput(openai, filePath) {
  return openai.files.create({
    file: fs.createReadStream(filePath),
    purpose: "batch",
  });
}

async function createBatch(openai, uploadedFileId, config, companyCount) {
  return openai.batches.create({
    input_file_id: uploadedFileId,
    endpoint: "/v1/responses",
    completion_window: "24h",
    metadata: {
      script: "batch-company-assessments",
      model: String(config.model),
      company_count: String(companyCount),
    },
  });
}

function parseOutputTextFromResponseBody(body) {
  if (!body || typeof body !== "object") return null;

  if (typeof body.output_text === "string" && body.output_text.trim()) {
    return body.output_text;
  }

  const parts = [];
  const output = Array.isArray(body.output) ? body.output : [];

  for (const item of output) {
    const content = Array.isArray(item?.content) ? item.content : [];
    for (const block of content) {
      if (typeof block?.text === "string") parts.push(block.text);
      else if (typeof block?.output_text === "string") parts.push(block.output_text);
    }
  }

  const text = parts.join("\n").trim();
  return text || null;
}

function parseBatchOutputJsonl(text) {
  const results = [];
  const lines = text.split(/\r?\n/).map((x) => x.trim()).filter(Boolean);

  for (const line of lines) {
    try {
      const row = JSON.parse(line);
      const responseBody = row?.response?.body ?? null;

      results.push({
        custom_id: row.custom_id ?? null,
        request_id: row.id ?? null,
        response_status: row?.response?.status_code ?? null,
        error: row?.error ?? null,
        output_text: parseOutputTextFromResponseBody(responseBody),
        response_body: responseBody,
      });
    } catch {
      results.push({
        parse_error: "Invalid JSONL line",
        raw_line: line,
      });
    }
  }

  return results;
}

async function saveRemoteFile(openai, fileId, outputPath) {
  const res = await openai.files.content(fileId);
  const text = await res.text();
  fs.writeFileSync(outputPath, text, "utf8");
  return text;
}

function updateBatchTimestamps(record, liveBatch) {
  const unixToIso = (value) => (value ? new Date(value * 1000).toISOString() : null);

  return {
    ...record,
    status: liveBatch.status,
    outputFileId: liveBatch.output_file_id ?? record.outputFileId ?? null,
    errorFileId: liveBatch.error_file_id ?? record.errorFileId ?? null,
    requestCounts: liveBatch.request_counts ?? record.requestCounts ?? null,
    lastCheckedAt: nowIso(),
    inProgressAt: unixToIso(liveBatch.in_progress_at) ?? record.inProgressAt ?? null,
    finalizingAt: unixToIso(liveBatch.finalizing_at) ?? record.finalizingAt ?? null,
    completedAt: unixToIso(liveBatch.completed_at) ?? record.completedAt ?? null,
    failedAt: unixToIso(liveBatch.failed_at) ?? record.failedAt ?? null,
    expiredAt: unixToIso(liveBatch.expired_at) ?? record.expiredAt ?? null,
    cancelledAt: unixToIso(liveBatch.cancelled_at) ?? record.cancelledAt ?? null,
  };
}

async function refreshOneBatch(openai, state, record) {
  const liveBatch = await openai.batches.retrieve(record.batchId);
  let updated = updateBatchTimestamps(record, liveBatch);

  if (updated.outputFileId && !updated.outputDownloaded) {
    const outputFileName = `batch-output-${updated.batchId}.jsonl`;
    const outputFilePath = path.join(__dirname, outputFileName);
    const outputText = await saveRemoteFile(openai, updated.outputFileId, outputFilePath);

    const parsedFileName = `batch-results-${updated.batchId}.json`;
    const parsedFilePath = path.join(__dirname, parsedFileName);
    writeJson(parsedFilePath, parseBatchOutputJsonl(outputText));

    updated = {
      ...updated,
      outputFileName,
      outputFilePath,
      outputDownloaded: true,
      parsedResultsFileName: parsedFileName,
      parsedResultsFilePath: parsedFilePath,
      parsedResultsWritten: true,
    };
  }

  if (updated.errorFileId && !updated.errorDownloaded) {
    const errorFileName = `batch-errors-${updated.batchId}.jsonl`;
    const errorFilePath = path.join(__dirname, errorFileName);
    await saveRemoteFile(openai, updated.errorFileId, errorFilePath);

    updated = {
      ...updated,
      errorFileName,
      errorFilePath,
      errorDownloaded: true,
    };
  }

  upsertBatch(state, updated);
  saveState(state);

  return updated;
}

function printBatch(record) {
  const counts = record.requestCounts || {};
  console.log(`Batch        : ${record.batchId}`);
  console.log(`Status       : ${record.status}`);
  console.log(`Model        : ${record.model}`);
  console.log(`Companies    : ${record.companyCount}`);
  console.log(`Completed    : ${counts.completed ?? 0}`);
  console.log(`Failed       : ${counts.failed ?? 0}`);
  console.log(`Total        : ${counts.total ?? 0}`);
  if (record.outputFileName) console.log(`Output file  : ${record.outputFileName}`);
  if (record.parsedResultsFileName) console.log(`Results file : ${record.parsedResultsFileName}`);
  if (record.errorFileName) console.log(`Errors file  : ${record.errorFileName}`);
  console.log("");
}

async function refreshKnownBatches(openai, state) {
  if (!state.batches.length) {
    console.log("No persisted batches found.\n");
    return;
  }

  console.log(`Checking ${state.batches.length} batch(es)...\n`);

  for (const record of [...state.batches]) {
    try {
      const updated = await refreshOneBatch(openai, state, record);
      printBatch(updated);
    } catch (error) {
      console.error(`Failed to refresh ${record.batchId}: ${error.message}\n`);
    }
  }
}

async function submitNewCompanies(openai, state, companies, source, config) {
  const newCompanies = getNewCompanies(state, companies);

  if (!newCompanies.length) {
    console.log("No new companies to submit.\n");
    return;
  }

  const { fileName, filePath } = writeBatchInputFile(newCompanies, config);
  const uploaded = await uploadBatchInput(openai, filePath);
  const batch = await createBatch(openai, uploaded.id, config, newCompanies.length);

  const batchRecord = {
    batchId: batch.id,
    status: batch.status,
    endpoint: batch.endpoint,
    inputFileId: uploaded.id,
    outputFileId: batch.output_file_id ?? null,
    errorFileId: batch.error_file_id ?? null,
    inputSource: source,
    inputFileName: fileName,
    inputFilePath: filePath,
    companyCount: newCompanies.length,
    companies: newCompanies,
    model: config.model,
    prompt: config.prompt,
    reasoning: buildReasoning(config) ?? null,
    maxOutputTokens: config.maxOutputTokens ?? null,
    submittedAt: nowIso(),
    lastCheckedAt: nowIso(),
    requestCounts: batch.request_counts ?? null,
    outputDownloaded: false,
    errorDownloaded: false,
    parsedResultsWritten: false,
  };

  upsertBatch(state, batchRecord);
  registerSubmittedCompanies(state, newCompanies, batch.id);
  saveState(state);

  console.log(`Submitted batch : ${batch.id}`);
  console.log(`New companies   : ${newCompanies.length}`);
  console.log(`Input file      : ${fileName}`);
  console.log(`Model           : ${config.model}`);
  console.log(
    `Thinking        : ${
      batchRecord.reasoning
        ? JSON.stringify(batchRecord.reasoning)
        : "model default"
    }`
  );
  console.log("");
}

async function main() {
  const args = parseArgs(process.argv.slice(2));

  if (args.help) {
    printHelp();
    return;
  }

  const fileConfig = loadConfig(args.config);
  const config = resolveConfig(args, fileConfig);
  const companyInput = loadCompanies(config.inputFile);
  const state = loadState();

  const openai = new OpenAI({ apiKey: config.apiKey });

  await refreshKnownBatches(openai, state);

  if (args.statusOnly) return;
  if (!args.submit) return;

  await submitNewCompanies(
    openai,
    state,
    companyInput.companies,
    companyInput.source,
    config
  );
}

main().catch((error) => {
  console.error("\nFatal error:");
  console.error(error?.stack || error?.message || String(error));
  process.exit(1);
});