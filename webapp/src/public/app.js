import { el, mount, setChildren } from "/vendor/redom.es.min.js";
import { marked } from "/vendor-marked/marked.esm.js";
import {
  DEFAULT_MODEL,
  MODEL_SUGGESTIONS,
  getReasoningProfile,
  normalizeModelName,
  normalizeReasoningEffort
} from "/shared/openai-models.js";

const BASE_NAV_ITEMS = [{ path: "/", label: "Welcome", icon: "welcome" }];
const PLATFORM_NAV_ITEMS = [
  { path: "/platform", label: "Platform", icon: "platform" },
  { path: "/platform/companies", label: "Companies", icon: "company" },
  { path: "/platform/reviews", label: "Reviews", icon: "review" },
  { path: "/platform/workflow-runs", label: "Workflow Runs", icon: "workflow" },
  { path: "/platform/ai-batch-jobs", label: "AI Batch Jobs", icon: "aijob" },
  { path: "/platform/capital-allocations", label: "Allocations", icon: "allocation" },
  { path: "/platform/positions", label: "Positions", icon: "portfolio" },
  { path: "/platform/config", label: "Config", icon: "config" },
  { path: "/platform/overrides", label: "Overrides", icon: "override" }
];
const BATCH_NAV_ITEMS = [
  { path: "/submissions", label: "Submissions", icon: "submissions" },
  { path: "/templated-submissions", label: "Templated Submissions", icon: "templated" },
  { path: "/procedures", label: "Procedures", icon: "procedures" },
  { path: "/procedure-executions", label: "Procedure Executions", icon: "executions" }
];
const ICONS = {
  welcome: "fa-solid fa-house",
  platform: "fa-solid fa-chart-pie",
  company: "fa-solid fa-building",
  review: "fa-solid fa-clipboard-check",
  workflow: "fa-solid fa-arrows-spin",
  aijob: "fa-solid fa-boxes-stacked",
  allocation: "fa-solid fa-wallet",
  portfolio: "fa-solid fa-briefcase",
  config: "fa-solid fa-sliders",
  override: "fa-solid fa-user-pen",
  thesis: "fa-solid fa-lightbulb",
  history: "fa-solid fa-clock-rotate-left",
  score: "fa-solid fa-gauge-high",
  search: "fa-solid fa-magnifying-glass",
  capital: "fa-solid fa-indian-rupee-sign",
  bucket: "fa-solid fa-layer-group",
  submissions: "fa-solid fa-inbox",
  templated: "fa-solid fa-table-list",
  procedures: "fa-solid fa-list-check",
  executions: "fa-solid fa-bolt",
  details: "fa-solid fa-circle-info",
  error: "fa-solid fa-circle-exclamation",
  raw: "fa-solid fa-code",
  copy: "fa-solid fa-copy",
  check: "fa-solid fa-check",
  menu: "fa-solid fa-bars",
  refresh: "fa-solid fa-rotate",
  add: "fa-solid fa-plus",
  edit: "fa-solid fa-pen",
  save: "fa-solid fa-floppy-disk",
  close: "fa-solid fa-xmark",
  back: "fa-solid fa-arrow-left",
  start: "fa-solid fa-play",
  open: "fa-solid fa-arrow-up-right-from-square",
  remove: "fa-solid fa-trash",
  files: "fa-solid fa-paperclip",
  prompt: "fa-solid fa-comment-dots",
  input: "fa-solid fa-arrow-right-to-bracket",
  output: "fa-solid fa-arrow-right-from-bracket",
  model: "fa-solid fa-microchip",
  thinking: "fa-solid fa-brain",
  procedure: "fa-solid fa-diagram-project",
  execution: "fa-solid fa-bolt",
  steps: "fa-solid fa-list-ol"
};

function icon(iconName, extraClassName = "") {
  const iconClassName = ICONS[iconName] || iconName;
  return el(`i${iconClassName ? `.${iconClassName.split(" ").join(".")}` : ""}${extraClassName ? `.${extraClassName.split(" ").join(".")}` : ""}`, {
    "aria-hidden": "true"
  });
}

function textWithIcon(iconName, text, className = "icon-text") {
  return el(`span.${className}`, icon(iconName, "inline-icon"), el("span", text));
}

function formatReasoningLabel(value) {
  if (!value) {
    return "N/A";
  }

  if (value === "xhigh") {
    return "X-High";
  }

  return value.charAt(0).toUpperCase() + value.slice(1);
}

function getModelOptions(selectedModel) {
  const resolvedModel = normalizeModelName(selectedModel);
  const options = MODEL_SUGGESTIONS.includes(resolvedModel)
    ? MODEL_SUGGESTIONS
    : [resolvedModel, ...MODEL_SUGGESTIONS];

  return options.map((model) => el("option", { value: model }, model));
}

function getAutomaticTheme(date = new Date()) {
  const hour = date.getHours();
  return hour >= 18 || hour < 8 ? "dark" : "light";
}

function applyTheme(theme) {
  document.documentElement.dataset.theme = theme;
}

function setButtonLabel(button, label, loading = false, iconName = button.dataset.icon || "") {
  button.dataset.label = label;
  button.dataset.icon = iconName || "";
  button.classList.toggle("button-loading", loading);

  if (loading) {
    setChildren(button, [el("span.button-spinner", { "aria-hidden": "true" }), el("span.button-label", label)]);
  } else {
    const children = [];

    if (iconName) {
      children.push(icon(iconName, "button-icon"));
    }

    children.push(el("span.button-label", label));
    setChildren(button, children);
  }
}

function submissionTypeLabel(type) {
  if (type === "templated") {
    return "Templated";
  }

  if (type === "procedure_execution") {
    return "Procedure Step";
  }

  return "Manual";
}

function iterationKindLabel(kind) {
  if (kind === "tool_result") {
    return "Tool Outputs";
  }

  return "Initial Request";
}

function statusClassName(status) {
  if (status === "completed") {
    return "status-completed";
  }

  if (["failed", "expired", "cancelled", "submission_failed"].includes(status)) {
    return "status-failed";
  }

  return "status-active";
}

function formatJson(value) {
  return JSON.stringify(value, null, 2);
}

const dateTimeFormatter = new Intl.DateTimeFormat("en-IN", {
  dateStyle: "medium",
  timeStyle: "short"
});

const currencyFormatter = new Intl.NumberFormat("en-IN", {
  style: "currency",
  currency: "INR",
  maximumFractionDigits: 0
});

const integerFormatter = new Intl.NumberFormat("en-IN");

function humanizeToken(value, fallback = "N/A") {
  if (!value && value !== 0) {
    return fallback;
  }

  return String(value)
    .replace(/_/g, " ")
    .replace(/\b\w/g, (match) => match.toUpperCase());
}

function formatDateTime(value, fallback = "N/A") {
  if (!value) {
    return fallback;
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return fallback;
  }

  return dateTimeFormatter.format(date);
}

function formatNumber(value, fallback = "N/A") {
  if (typeof value !== "number" || Number.isNaN(value)) {
    return fallback;
  }

  return integerFormatter.format(value);
}

function formatCurrency(value, fallback = "N/A") {
  if (typeof value !== "number" || Number.isNaN(value)) {
    return fallback;
  }

  return currencyFormatter.format(value);
}

function formatPercent(value, fallback = "N/A") {
  if (typeof value !== "number" || Number.isNaN(value)) {
    return fallback;
  }

  return `${value.toFixed(value >= 10 ? 0 : 1)}%`;
}

function formatScore(value, fallback = "N/A") {
  if (typeof value !== "number" || Number.isNaN(value)) {
    return fallback;
  }

  return value.toFixed(1);
}

function infoRow(label, value) {
  return el("div", el("dt", label), el("dd", value));
}

function listOrEmpty(items, emptyText = "No items yet.") {
  if (!Array.isArray(items) || !items.length) {
    return el(".empty-state", el("p", emptyText));
  }

  return el("ul.detail-bullet-list", ...items.map((item) => el("li", item)));
}

function resultJsonBlock(value, emptyText = "No data available.") {
  if (!value) {
    return el(".empty-state", el("p", emptyText));
  }

  return buildCopyableSurface(el(".result-box", el("pre", formatJson(value))), () => formatJson(value));
}

async function copyTextToClipboard(text) {
  const resolvedText = String(text ?? "");

  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(resolvedText);
    return;
  }

  const fallback = el("textarea", {
    value: resolvedText,
    readonly: true
  });

  fallback.style.position = "fixed";
  fallback.style.opacity = "0";
  fallback.style.pointerEvents = "none";
  document.body.appendChild(fallback);
  fallback.focus();
  fallback.select();
  document.execCommand("copy");
  fallback.remove();
}

function createCopyButton(getText, label = "Copy", options = {}) {
  const button = el(
    "button.button.button-secondary.button-small.copy-button",
    {
      type: "button",
      onclick: async (event) => {
        if (options.stopToggle) {
          event.preventDefault();
          event.stopPropagation();
        }

        const text = typeof getText === "function" ? getText() : getText;
        window.clearTimeout(button.copyResetTimer);

        try {
          await copyTextToClipboard(text);
          setButtonLabel(button, "Copied", false, "check");
        } catch (error) {
          setButtonLabel(button, "Copy Failed", false, "error");
        }

        button.copyResetTimer = window.setTimeout(() => {
          setButtonLabel(button, label, false, "copy");
        }, 1400);
      }
    },
    label
  );

  setButtonLabel(button, label, false, "copy");
  return button;
}

function buildCopyableSurface(contentNode, getText) {
  return el(".copy-surface", el(".copy-surface-actions", createCopyButton(getText)), contentNode);
}

function buildCollapsibleCopySection({ iconName, label, contentNode, getText, open = false }) {
  const disclosure = el("details.raw-disclosure.submission-iteration-disclosure");
  disclosure.open = open;

  setChildren(disclosure, [
    el(
      "summary.raw-summary.submission-iteration-disclosure-summary",
      el(".submission-iteration-disclosure-title", textWithIcon(iconName, label)),
      createCopyButton(getText, "Copy", { stopToggle: true })
    ),
    el(".submission-iteration-disclosure-body", contentNode)
  ]);

  return disclosure;
}

async function apiFetch(url, options = {}) {
  const isFormData = typeof FormData !== "undefined" && options.body instanceof FormData;
  const response = await fetch(url, {
    headers: isFormData
      ? {
          ...(options.headers || {})
        }
      : {
          "Content-Type": "application/json",
          ...(options.headers || {})
        },
    ...options
  });

  const contentType = response.headers.get("content-type") || "";
  const payload = contentType.includes("application/json") ? await response.json() : null;

  if (!response.ok) {
    const error = new Error(payload?.error || "Request failed.");
    error.status = response.status;
    throw error;
  }

  return payload;
}

async function apiFetchOptional(url, options = {}) {
  try {
    return await apiFetch(url, options);
  } catch (error) {
    if (error.status === 404) {
      return null;
    }

    throw error;
  }
}

function pathForSubmission(id) {
  return `/submissions/${id}`;
}

function pathForExecution(id) {
  return `/procedure-executions/${id}`;
}

function pathForPlatformCompany(id) {
  return `/platform/companies/${id}`;
}

function pathForPlatformReview(id) {
  return `/platform/reviews/${id}`;
}

function pathForPlatformWorkflowRun(id) {
  return `/platform/workflow-runs/${id}`;
}

function pathForPlatformAIBatchJob(id) {
  return `/platform/ai-batch-jobs/${id}`;
}

function pathForPlatformCapitalAllocation(id) {
  return `/platform/capital-allocations/${id}`;
}

function pathForPlatformConfigSnapshot(id) {
  return `/platform/config/snapshots/${id}`;
}

function getRoute() {
  const pathname = window.location.pathname;
  const submissionMatch = pathname.match(/^\/submissions\/([^/]+)$/);
  const executionMatch = pathname.match(/^\/procedure-executions\/([^/]+)$/);
  const platformCompanyMatch = pathname.match(/^\/platform\/companies\/([^/]+)$/);
  const platformReviewMatch = pathname.match(/^\/platform\/reviews\/([^/]+)$/);
  const platformWorkflowMatch = pathname.match(/^\/platform\/workflow-runs\/([^/]+)$/);
  const platformAIBatchJobMatch = pathname.match(/^\/platform\/ai-batch-jobs\/([^/]+)$/);
  const platformAllocationMatch = pathname.match(/^\/platform\/capital-allocations\/([^/]+)$/);
  const platformConfigSnapshotMatch = pathname.match(/^\/platform\/config\/snapshots\/([^/]+)$/);

  if (submissionMatch) {
    return {
      page: "submission-detail",
      submissionId: submissionMatch[1]
    };
  }

  if (executionMatch) {
    return {
      page: "execution-detail",
      executionId: executionMatch[1]
    };
  }

  if (platformCompanyMatch) {
    return {
      page: "platform-company-detail",
      companyId: platformCompanyMatch[1]
    };
  }

  if (platformReviewMatch) {
    return {
      page: "platform-review-detail",
      reviewId: platformReviewMatch[1]
    };
  }

  if (platformWorkflowMatch) {
    return {
      page: "platform-workflow-run-detail",
      workflowRunId: platformWorkflowMatch[1]
    };
  }

  if (platformAIBatchJobMatch) {
    return {
      page: "platform-ai-batch-job-detail",
      aiBatchJobId: platformAIBatchJobMatch[1]
    };
  }

  if (platformAllocationMatch) {
    return {
      page: "platform-capital-allocation-detail",
      capitalAllocationId: platformAllocationMatch[1]
    };
  }

  if (platformConfigSnapshotMatch) {
    return {
      page: "platform-config-snapshot-detail",
      configSnapshotId: platformConfigSnapshotMatch[1]
    };
  }

  if (pathname === "/submissions") {
    return { page: "submissions", submissionId: null };
  }

  if (pathname === "/templated-submissions") {
    return { page: "templated-submissions", submissionId: null };
  }

  if (pathname === "/procedures") {
    return { page: "procedures", submissionId: null };
  }

  if (pathname === "/procedure-executions") {
    return { page: "procedure-executions", submissionId: null };
  }

  if (pathname === "/platform") {
    return { page: "platform-home" };
  }

  if (pathname === "/platform/companies") {
    return { page: "platform-companies" };
  }

  if (pathname === "/platform/reviews") {
    return { page: "platform-reviews" };
  }

  if (pathname === "/platform/workflow-runs") {
    return { page: "platform-workflow-runs" };
  }

  if (pathname === "/platform/ai-batch-jobs") {
    return { page: "platform-ai-batch-jobs" };
  }

  if (pathname === "/platform/capital-allocations") {
    return { page: "platform-capital-allocations" };
  }

  if (pathname === "/platform/positions") {
    return { page: "platform-positions" };
  }

  if (pathname === "/platform/config") {
    return { page: "platform-config" };
  }

  if (pathname === "/platform/overrides") {
    return { page: "platform-overrides" };
  }

  return { page: "welcome", submissionId: null };
}

function isPlatformRoutePage(page) {
  return typeof page === "string" && page.startsWith("platform");
}

function isBatchRoutePage(page) {
  return [
    "submissions",
    "submission-detail",
    "templated-submissions",
    "procedures",
    "procedure-executions",
    "execution-detail"
  ].includes(page);
}

function primaryPathForRoute(route) {
  switch (route.page) {
    case "submission-detail":
      return "/submissions";
    case "execution-detail":
      return "/procedure-executions";
    case "platform-company-detail":
      return "/platform/companies";
    case "platform-review-detail":
      return "/platform/reviews";
    case "platform-workflow-run-detail":
      return "/platform/workflow-runs";
    case "platform-ai-batch-job-detail":
      return "/platform/ai-batch-jobs";
    case "platform-capital-allocation-detail":
      return "/platform/capital-allocations";
    case "platform-config-snapshot-detail":
      return "/platform/config";
    default:
      return window.location.pathname;
  }
}

function safeArray(value) {
  return Array.isArray(value) ? value : [];
}

function firstNonEmpty(...values) {
  for (const value of values) {
    if (typeof value === "string" && value.trim()) {
      return value;
    }
  }

  return "";
}

function formatDurationMs(value, fallback = "N/A") {
  if (typeof value !== "number" || Number.isNaN(value) || value < 0) {
    return fallback;
  }

  if (value >= 1000) {
    return `${(value / 1000).toFixed(value >= 10000 ? 0 : 1)}s`;
  }

  return `${value} ms`;
}

function formatBool(value, trueLabel = "Yes", falseLabel = "No") {
  return value ? trueLabel : falseLabel;
}

function createBadge(text, className = "type-pill") {
  return el("span", { className }, text);
}

function createStatusPill(status) {
  return el("span", { className: `status-pill ${statusClassName(status)}` }, humanizeToken(status));
}

function createActionButton({ label, iconName, onclick, variant = "secondary", small = false, type = "button" }) {
  const button = el("button", {
    type,
    className: `button button-${variant}${small ? " button-small" : ""}`,
    onclick
  }, label);
  setButtonLabel(button, label, false, iconName);
  return button;
}

function createNavigateButton(app, path, label, iconName, options = {}) {
  return createActionButton({
    label,
    iconName,
    variant: options.variant || "secondary",
    small: options.small || false,
    onclick: () => app.navigate(path)
  });
}

function buildMetricCard(label, value, note = "") {
  return el(
    ".metric-card",
    el("p.metric-label", label),
    el("strong.metric-value", value),
    note ? el("p.metric-note", note) : null
  );
}

function buildJSONSection(iconName, title, value, emptyText = "No data available.") {
  return el(".detail-block", el("h2", textWithIcon(iconName, title)), resultJsonBlock(value, emptyText));
}

function buildInlineBadgeRow(items) {
  const badges = items.filter(Boolean);
  if (!badges.length) {
    return el(".inline-badges");
  }

  return el(".inline-badges", ...badges);
}

async function detectCapability(url) {
  try {
    const response = await fetch(url, {
      headers: {
        Accept: "application/json"
      }
    });
    return response.ok;
  } catch (error) {
    return false;
  }
}

class NavLink {
  constructor(app, item) {
    this.app = app;
    this.item = item;
    this.el = el(
      "button.nav-link",
      {
        type: "button",
        onclick: () => this.app.navigate(item.path)
      },
      textWithIcon(item.icon, item.label, "nav-link-content")
    );
  }

  update(currentPath) {
    this.el.className = `nav-link${currentPath === this.item.path ? " nav-link-active" : ""}`;
  }
}

class Sidebar {
  constructor(app) {
    this.app = app;
    this.closeButton = el(
      "button.button.button-secondary.button-small.sidebar-close",
      {
        type: "button",
        onclick: () => this.app.closeSidebar()
      },
      "Close"
    );
    setButtonLabel(this.closeButton, "Close", false, "close");
    this.nav = el("nav.sidebar-nav");
    this.el = el(
      "aside.sidebar",
      el(
        ".sidebar-header",
        el("div", el("p.eyebrow", textWithIcon("procedure", "Workspace")), el("h2.sidebar-title", "Selvam Workspace")),
        this.closeButton
      ),
      this.nav
    );
  }

  update(currentPath, isOpen, items) {
    this.links = items.map((item) => new NavLink(this.app, item));
    this.links.forEach((link) => link.update(currentPath));
    setChildren(this.nav, this.links);
    this.el.className = `sidebar${isOpen ? " sidebar-open" : ""}`;
  }
}

class ModelSettingsControl {
  constructor({ model = DEFAULT_MODEL, reasoningEffort = "" } = {}) {
    this.isBusy = false;
    this.modelSelect = el("select.select-control", {
      onchange: () => this.syncReasoningOptions()
    });
    this.reasoningSelect = el("select.select-control");
    this.el = el(
      ".compact-field-grid",
      el(".field-block.field-block-compact", el("label.label", textWithIcon("model", "Model")), this.modelSelect),
      el(".field-block.field-block-compact", el("label.label", textWithIcon("thinking", "Thinking")), this.reasoningSelect)
    );
    this.syncModelOptions(model);
    this.syncReasoningOptions(reasoningEffort);
  }

  syncModelOptions(selectedModel) {
    setChildren(this.modelSelect, getModelOptions(selectedModel));
    this.modelSelect.value = normalizeModelName(selectedModel);
    this.modelSelect.disabled = this.isBusy;
  }

  syncReasoningOptions(preferredValue) {
    const profile = getReasoningProfile(this.modelSelect.value);

    if (!profile.supported) {
      setChildren(this.reasoningSelect, [el("option", { value: "" }, "N/A")]);
      this.reasoningSelect.value = "";
      this.reasoningSelect.disabled = true;
      return;
    }

    setChildren(
      this.reasoningSelect,
      profile.options.map((value) => el("option", { value }, formatReasoningLabel(value)))
    );
    this.reasoningSelect.value = normalizeReasoningEffort(this.modelSelect.value, preferredValue ?? this.reasoningSelect.value);
    this.reasoningSelect.disabled = this.isBusy;
  }

  getModel() {
    return this.modelSelect.value || DEFAULT_MODEL;
  }

  getReasoningEffort() {
    return this.reasoningSelect.disabled && this.reasoningSelect.value === "" ? "" : this.reasoningSelect.value;
  }

  setBusy(isBusy) {
    this.isBusy = isBusy;
    this.modelSelect.disabled = isBusy;
    this.syncReasoningOptions();
  }
}

class PromptField {
  constructor(entry, onRemove) {
    this.onRemove = onRemove;
    this.files = [];
    this.index = 0;
    this.canRemove = true;
    this.isBusy = false;
    this.label = el("label.label");
    this.copyButton = createCopyButton(() => this.textarea.value);
    this.removeButton = el(
      "button.button.button-secondary.button-small.prompt-remove",
      {
        type: "button",
        onclick: () => this.onRemove(this)
      },
      "Remove"
    );
    setButtonLabel(this.removeButton, "Remove", false, "remove");
    this.fileInput = el("input.prompt-file-input", {
      type: "file",
      multiple: true,
      onchange: (event) => {
        this.addFiles(Array.from(event.target.files || []));
        event.target.value = "";
      }
    });
    this.fileList = el(".selected-file-list");
    this.textarea = el("textarea", {
      rows: 5,
      placeholder: "Add a prompt or leave this blank and attach files."
    });
    this.textarea.value = entry?.query || "";
    this.modelSettings = new ModelSettingsControl({
      model: entry?.model || DEFAULT_MODEL,
      reasoningEffort: entry?.reasoningEffort || ""
    });
    this.addFilesButton = el(
      "button.button.button-secondary.button-small.file-picker-button",
      {
        type: "button",
        onclick: () => this.fileInput.click()
      },
      "Add Files"
    );
    setButtonLabel(this.addFilesButton, "Add Files", false, "files");
    this.filesSection = el(
      ".prompt-files-section",
      el(".prompt-files-header", el("span.label", textWithIcon("files", "Files")), this.addFilesButton),
      this.fileList,
      this.fileInput
    );
    this.el = el(
      ".prompt-card",
      el(".prompt-card-header", this.label, el(".inline-actions", this.copyButton, this.removeButton)),
      this.textarea,
      this.modelSettings.el,
      this.filesSection
    );
    this.update({ index: 0, canRemove: true });
    this.renderFiles();
  }

  update({ index, canRemove }) {
    this.index = index;
    this.canRemove = canRemove;
    this.label.textContent = `Prompt ${index + 1}`;
    this.textarea.id = `prompt-${index}`;
    this.label.htmlFor = this.textarea.id;
    this.removeButton.disabled = this.isBusy || !canRemove;
  }

  getValue() {
    return this.textarea.value;
  }

  getModel() {
    return this.modelSettings.getModel();
  }

  getReasoningEffort() {
    return this.modelSettings.getReasoningEffort();
  }

  getFiles() {
    return this.files;
  }

  hasContent() {
    return Boolean(this.getValue().trim() || this.files.length);
  }

  addFiles(files) {
    const existingKeys = new Set(this.files.map((file) => `${file.name}:${file.size}:${file.lastModified}`));

    for (const file of files) {
      const key = `${file.name}:${file.size}:${file.lastModified}`;

      if (!existingKeys.has(key)) {
        this.files.push(file);
        existingKeys.add(key);
      }
    }

    this.renderFiles();
  }

  removeFile(fileToRemove) {
    this.files = this.files.filter((file) => file !== fileToRemove);
    this.renderFiles();
  }

  renderFiles() {
    const items = this.files.map((file) =>
      el(
        ".selected-file-item",
        el(".selected-file-meta", el("span.selected-file-name", file.name), el("span.selected-file-size", `${Math.max(1, Math.round(file.size / 1024))} KB`)),
        el(
          "button.icon-button",
          {
            type: "button",
            disabled: this.isBusy,
            onclick: () => this.removeFile(file)
          },
          textWithIcon("remove", "Delete")
        )
      )
    );

    if (!items.length) {
      items.push(el("p.muted.selected-file-empty", "No files added."));
    }

    setChildren(this.fileList, items);
  }

  setBusy(isBusy) {
    this.isBusy = isBusy;
    this.update({
      index: this.index,
      canRemove: this.canRemove
    });
    this.textarea.disabled = isBusy;
    this.copyButton.disabled = isBusy;
    this.fileInput.disabled = isBusy;
    this.addFilesButton.disabled = isBusy;
    this.modelSettings.setBusy(isBusy);
    this.renderFiles();
  }
}

class ManualSubmissionForm {
  constructor(app) {
    this.app = app;
    this.fields = [];
    this.isBusy = false;
    this.stack = el(".prompt-stack");
    this.error = el("p.form-error");
    this.addButton = el(
      "button.button.button-secondary",
      {
        type: "button",
        onclick: () => this.addField()
      },
      "Add Another Prompt"
    );
    this.submitButton = el("button.button.button-primary", { type: "submit" }, "Submit All Prompts");
    setButtonLabel(this.addButton, "Add Another Prompt", false, "add");
    setButtonLabel(this.submitButton, "Submit All Prompts", false, "save");
    this.form = el(
      "form.query-form",
      {
        onsubmit: async (event) => {
          event.preventDefault();
          await this.submit();
        }
      },
      this.stack,
      this.error,
      el(".prompt-actions", this.addButton, this.submitButton)
    );
    this.el = el(
      ".panel",
      el(
        ".section-heading",
        el("h2", textWithIcon("prompt", "New Queries")),
        el("p", "Add one or many prompts, then submit them together as one OpenAI batch.")
      ),
      this.form
    );

    this.addField();
    this.setError("");
  }

  addField(entry = {}) {
    const field = new PromptField(entry, (target) => this.removeField(target));
    this.fields.push(field);
    this.syncFields();
  }

  removeField(field) {
    if (this.fields.length === 1) {
      return;
    }

    this.fields = this.fields.filter((item) => item !== field);
    this.syncFields();
  }

  syncFields() {
    this.fields.forEach((field, index) => {
      field.update({
        index,
        canRemove: this.fields.length > 1
      });
      field.setBusy(this.isBusy);
    });

    setChildren(this.stack, this.fields);
  }

  getPrompts() {
    return this.fields.map((field) => field.getValue());
  }

  buildFormData() {
    const formData = new FormData();
    const payload = this.fields.map((field) => ({
      query: field.getValue(),
      model: field.getModel(),
      reasoningEffort: field.getReasoningEffort()
    }));

    formData.append("submissionPayload", JSON.stringify(payload));

    this.fields.forEach((field, index) => {
      formData.append("prompts", field.getValue());

      for (const file of field.getFiles()) {
        formData.append(`promptFiles-${index}`, file);
      }
    });

    return formData;
  }

  reset() {
    this.fields = [];
    this.addField();
    this.setError("");
  }

  setError(message) {
    this.error.textContent = message || "";
    this.error.style.display = message ? "" : "none";
  }

  setBusy(isBusy) {
    this.isBusy = isBusy;
    this.addButton.disabled = isBusy;
    this.submitButton.disabled = isBusy;
    this.fields.forEach((field) => field.setBusy(isBusy));
  }

  async submit() {
    try {
      this.setError("");
      const formData = this.buildFormData();
      const hasAtLeastOneSubmission = this.fields.some((field) => field.hasContent());

      if (!hasAtLeastOneSubmission) {
        this.setError("Please add at least one prompt or file before submitting.");
        return;
      }

      this.app.setBusy(true);
      setButtonLabel(this.submitButton, "Submit All Prompts", true, "save");
      const payload = await apiFetch("/api/submissions", {
        method: "POST",
        body: formData
      });

      this.reset();
      this.app.setMessage(`Submitted ${payload.jobs.length} prompt${payload.jobs.length === 1 ? "" : "s"} successfully.`);
      await this.app.loadSubmissions();
      this.app.navigate("/submissions");
    } catch (error) {
      this.setError(error.message);
    } finally {
      setButtonLabel(this.submitButton, "Submit All Prompts", false, "save");
      this.app.setBusy(false);
    }
  }
}

class TemplatedSubmissionForm {
  constructor(app) {
    this.app = app;
    this.error = el("p.form-error");
    this.jsonInput = el("textarea", {
      rows: 10,
      placeholder: '[{"userId":"123"},{"userId":"456"}]'
    });
    this.jsonCopyButton = createCopyButton(() => this.jsonInput.value);
    this.templateInput = el("textarea", {
      rows: 8,
      placeholder: "Summarize the account for user {userId}."
    });
    this.templateCopyButton = createCopyButton(() => this.templateInput.value);
    this.modelSettings = new ModelSettingsControl({
      model: DEFAULT_MODEL
    });
    this.submitButton = el(
      "button.button.button-primary",
      {
        type: "submit"
      },
      "Build And Submit Batch"
    );
    setButtonLabel(this.submitButton, "Build And Submit Batch", false, "save");
    this.form = el(
      "form.query-form",
      {
        onsubmit: async (event) => {
          event.preventDefault();
          await this.submit();
        }
      },
      el(
        ".field-block",
        el(".field-label-row", el("label.label", { htmlFor: "records-json" }, textWithIcon("templated", "JSON Array")), this.jsonCopyButton),
        this.jsonInput
      ),
      el(
        ".field-block",
        el(".field-label-row", el("label.label", { htmlFor: "prompt-template" }, textWithIcon("prompt", "Prompt Template")), this.templateCopyButton),
        this.templateInput
      ),
      this.modelSettings.el,
      this.error,
      el(".prompt-actions", this.submitButton)
    );
    this.jsonInput.id = "records-json";
    this.templateInput.id = "prompt-template";
    this.el = el(
      ".panel",
      el(
        ".section-heading",
        el("h2", textWithIcon("templated", "Templated Submissions")),
        el("p", "Provide a JSON array and a prompt template. Each object becomes one prompt in a shared batch.")
      ),
      this.form
    );
    this.setError("");
  }

  setError(message) {
    this.error.textContent = message || "";
    this.error.style.display = message ? "" : "none";
  }

  setBusy(isBusy) {
    this.submitButton.disabled = isBusy;
    this.jsonInput.disabled = isBusy;
    this.jsonCopyButton.disabled = isBusy;
    this.templateInput.disabled = isBusy;
    this.templateCopyButton.disabled = isBusy;
    this.modelSettings.setBusy(isBusy);
  }

  async submit() {
    try {
      this.app.setBusy(true);
      this.setError("");
      setButtonLabel(this.submitButton, "Build And Submit Batch", true, "save");

      let records;

      try {
        records = JSON.parse(this.jsonInput.value || "[]");
      } catch (error) {
        throw new Error("JSON array input must be valid JSON.");
      }

      const payload = await apiFetch("/api/templated-submissions", {
        method: "POST",
        body: JSON.stringify({
          records,
          promptTemplate: this.templateInput.value,
          model: this.modelSettings.getModel(),
          reasoningEffort: this.modelSettings.getReasoningEffort()
        })
      });

      this.app.setMessage(`Created ${payload.jobs.length} templated prompt${payload.jobs.length === 1 ? "" : "s"} successfully.`);
      await this.app.loadSubmissions();
      this.app.navigate("/submissions");
    } catch (error) {
      this.setError(error.message);
    } finally {
      setButtonLabel(this.submitButton, "Build And Submit Batch", false, "save");
      this.app.setBusy(false);
    }
  }
}

class SubmissionCard {
  constructor(app) {
    this.app = app;
    this.status = el("span.status-pill");
    this.date = el("span.job-date");
    this.typeBadge = el("span.type-pill");
    this.titleLink = el(
      "a.job-link",
      {
        href: "/submissions",
        onclick: (event) => {
          event.preventDefault();
          this.app.navigate(pathForSubmission(this.job.id));
        }
      }
    );
    this.metaBatch = el("span");
    this.metaModel = el("span");
    this.metaReasoning = el("span");
    this.metaSync = el("span");
    this.actions = el(".job-actions");
    this.refreshButton = el(
      "button.button.button-secondary",
      {
        type: "button",
        onclick: async (event) => {
          await this.app.refreshSubmission(this.job.id, { stayOnDetail: false, button: event.currentTarget });
        }
      },
      "Refresh Status"
    );
    setButtonLabel(this.refreshButton, "Refresh", false, "refresh");
    this.el = el(
      "article.job-card",
      el(".job-header", this.status, this.date),
      el(".job-type-row", this.typeBadge),
      el("h3.job-title", this.titleLink),
      el(".job-meta", this.metaBatch, this.metaModel, this.metaReasoning, this.metaSync),
      this.actions
    );
  }

  update(job) {
    this.job = job;
    this.status.className = `status-pill ${statusClassName(job.status)}`;
    this.status.textContent = job.status;
    this.date.textContent = `Created ${job.createdAtLabel}`;
    this.typeBadge.textContent = submissionTypeLabel(job.submissionType);
    this.titleLink.href = pathForSubmission(job.id);
    this.titleLink.textContent = job.queryLabel;
    this.titleLink.title = job.queryLabel === "N/A" ? "No prompt provided" : job.query;
    this.metaBatch.innerHTML = `Batch: <code>${job.batchId || "pending"}</code>`;
    this.metaModel.textContent = `Model: ${job.model}`;
    this.metaReasoning.textContent = `Thinking: ${job.reasoningEffortLabel}`;
    this.metaSync.textContent = `Last sync: ${job.lastSyncedAtLabel}`;
    const actions = [];

    if (job.canRefresh) {
      actions.push(this.refreshButton);
    }

    setChildren(this.actions, actions);
    this.actions.style.display = actions.length ? "" : "none";
  }
}

class SubmissionsList {
  constructor(app) {
    this.app = app;
    this.count = el("p");
    this.cards = [];
    this.cardsHost = el(".job-list");
    this.empty = el(".empty-state", el("p", "No submissions yet. Start with a manual or templated batch."));
    this.el = el(
      ".panel",
      el(".section-heading", el("h2", textWithIcon("submissions", "Submissions")), this.count),
      this.empty,
      this.cardsHost
    );
  }

  update(jobs) {
    this.count.textContent = `${jobs.length} total`;
    this.empty.style.display = jobs.length ? "none" : "";
    this.cards = jobs.map((job, index) => {
      const card = this.cards[index] || new SubmissionCard(this.app);
      card.update(job);
      return card;
    });
    setChildren(this.cardsHost, this.cards);
  }
}

class SubmissionDetail {
  constructor(app) {
    this.app = app;
    this.topline = el(".detail-topline");
    this.title = el("h1.detail-title.detail-title-ellipsis", "Submission Detail");
    this.info = el(".detail-list");
    this.meta = el(".detail-block");
    this.template = el(".detail-block");
    this.files = el(".detail-block");
    this.iterations = el(".detail-block");
    this.el = el(
      ".panel.detail-panel",
      this.topline,
      this.title,
      this.meta,
      this.template,
      this.files,
      this.iterations
    );
  }

  infoRow(label, value) {
    return el("div", el("dt", label), el("dd", value));
  }

  buildIterationCard(job, iteration, isLatest = false) {
    const card = el("details.submission-iteration-card");
    card.open = isLatest || iteration.status !== "completed";

    const summary = el(
      "summary.submission-iteration-summary",
      el(
        ".submission-iteration-copy",
        el(".submission-iteration-title", `Iteration ${iteration.iterationNumber}`),
        el(
          ".submission-iteration-meta",
          `${iterationKindLabel(iteration.kind)} • Batch ${iteration.batchId || "pending"} • ${
            iteration.completedAtLabel || iteration.lastSyncedAtLabel
          }`
        )
      ),
      el(
        ".submission-iteration-status",
        el("span.type-pill", iterationKindLabel(iteration.kind)),
        el("span.status-pill", { className: `status-pill ${statusClassName(iteration.status)}` }, iteration.status)
      )
    );

    const inputBox = buildCopyableSurface(
      el(".result-box", el("pre", iteration.inputTextLabel || "N/A")),
      () => iteration.inputText || ""
    );
    const inputDisclosure = buildCollapsibleCopySection({
      iconName: "input",
      label: "Input",
      contentNode: inputBox,
      getText: () => iteration.inputText || ""
    });

    const outputChildren = [];
    if (iteration.resultText) {
      const outputMarkdown = el(".markdown-output");
      outputMarkdown.innerHTML = marked.parse(iteration.resultText);
      outputChildren.push(buildCopyableSurface(el(".result-box.markdown-result", outputMarkdown), () => iteration.resultText));
    } else {
      outputChildren.push(
        el("p.muted", iteration.canRefresh ? "Waiting for this iteration to finish." : "No output text is available for this iteration.")
      );
    }

    if (isLatest && (job.canRefresh || iteration.canRefresh)) {
      const refreshButton = el(
        "button.button.button-secondary",
        {
          type: "button",
          onclick: async (event) => {
            await this.app.refreshSubmission(job.id, { stayOnDetail: true, button: event.currentTarget });
          }
        },
        "Refresh Status"
      );
      setButtonLabel(refreshButton, "Refresh", false, "refresh");
      outputChildren.push(refreshButton);
    }
    const outputDisclosure = buildCollapsibleCopySection({
      iconName: "output",
      label: "Output",
      contentNode: el(".submission-iteration-output-stack", ...outputChildren),
      getText: () => iteration.resultText || ""
    });

    const sections = [
      el(
        ".submission-iteration-section",
        inputDisclosure
      ),
      el(
        ".submission-iteration-section",
        outputDisclosure
      )
    ];

    if (Array.isArray(iteration.toolCalls) && iteration.toolCalls.length) {
      sections.push(
        el(
          ".submission-iteration-section",
          el("h3", textWithIcon("details", "Tool Calls")),
          buildCopyableSurface(el(".result-box", el("pre", formatJson(iteration.toolCalls))), () => formatJson(iteration.toolCalls))
        )
      );
    }

    if (Array.isArray(iteration.toolOutputs) && iteration.toolOutputs.length) {
      sections.push(
        el(
          ".submission-iteration-section",
          el("h3", textWithIcon("details", "Submitted Tool Outputs")),
          buildCopyableSurface(el(".result-box", el("pre", formatJson(iteration.toolOutputs))), () => formatJson(iteration.toolOutputs))
        )
      );
    }

    if (iteration.latestErrorLine?.error) {
      sections.push(
        el(
          ".submission-iteration-section",
          el("h3", textWithIcon("error", "Error")),
          buildCopyableSurface(el(".result-box.result-box-error", el("pre", formatJson(iteration.latestErrorLine.error))), () =>
            formatJson(iteration.latestErrorLine.error)
          )
        )
      );
    }

    if (iteration.requestBody || iteration.resultResponseBody) {
      const rawDisclosure = el("details.raw-disclosure");
      rawDisclosure.open = false;
      const rawBlocks = [];

      if (iteration.requestBody) {
        rawBlocks.push(
          el(
            ".submission-iteration-section",
            el("h3", textWithIcon("raw", "Raw Request")),
            buildCopyableSurface(el(".result-box", el("pre", formatJson(iteration.requestBody))), () => formatJson(iteration.requestBody))
          )
        );
      }

      if (iteration.resultResponseBody) {
        rawBlocks.push(
          el(
            ".submission-iteration-section",
            el("h3", textWithIcon("raw", "Raw Response")),
            buildCopyableSurface(
              el(".result-box", el("pre", formatJson(iteration.resultResponseBody))),
              () => formatJson(iteration.resultResponseBody)
            )
          )
        );
      }

      sections.push(
        el(
          ".submission-iteration-section",
          rawDisclosure
        )
      );
      setChildren(rawDisclosure, [
        el("summary.raw-summary", textWithIcon("raw", "Raw Payloads")),
        ...rawBlocks
      ]);
    }

    setChildren(card, [
      summary,
      el(".submission-iteration-content", ...sections)
    ]);

    return card;
  }

  update(job) {
    const backButton = el(
      "button.button.button-secondary.button-small",
      {
        type: "button",
        onclick: () => this.app.navigate("/submissions")
      },
      "Back to submissions"
    );
    setButtonLabel(backButton, "Back", false, "back");
    const status = el("span.status-pill", { className: `status-pill ${statusClassName(job.status)}` }, job.status);
    setChildren(this.topline, [backButton, status]);
    this.title.textContent = job.queryLabel === "N/A" ? "Submission Detail" : job.queryLabel;
    this.title.title = job.queryLabel === "N/A" ? "Submission Detail" : job.queryLabel;

    setChildren(this.info, [
      this.infoRow("Type", submissionTypeLabel(job.submissionType)),
      this.infoRow("Model", job.model),
      this.infoRow("Thinking", job.reasoningEffortLabel),
      this.infoRow("Iterations", String(job.iterationCount || (Array.isArray(job.iterations) ? job.iterations.length : 0))),
      this.infoRow("Batch ID", el("code", job.batchId || "pending")),
      this.infoRow("Created", job.createdAtLabel),
      this.infoRow("Last synced", job.lastSyncedAtLabel),
      this.infoRow("Completed", job.completedAtLabel || "Not yet")
    ]);
    setChildren(this.meta, [el("h2", textWithIcon("details", "Details")), this.info]);

    if (job.submissionType === "templated" && (job.promptTemplate || job.templateRecord)) {
      const blocks = [el("h2", textWithIcon("templated", "Template Context"))];

      if (job.promptTemplate) {
        blocks.push(buildCopyableSurface(el(".result-box", el("pre", job.promptTemplate)), () => job.promptTemplate));
      }

      if (job.templateRecord) {
        blocks.push(buildCopyableSurface(el(".result-box", el("pre", formatJson(job.templateRecord))), () => formatJson(job.templateRecord)));
      }

      setChildren(this.template, blocks);
      this.template.style.display = "";
    } else {
      this.template.style.display = "none";
      setChildren(this.template, []);
    }

    if (Array.isArray(job.attachedFiles) && job.attachedFiles.length) {
      setChildren(this.files, [
        el("h2", textWithIcon("files", "Attached Files")),
        el(
          ".detail-file-list",
          ...job.attachedFiles.map((file) =>
            el(
              ".detail-file-item",
              el("span.selected-file-name", file.originalName || "File"),
              el("span.selected-file-size", file.mimeType || "file")
            )
          )
        )
      ]);
      this.files.style.display = "";
    } else {
      this.files.style.display = "none";
      setChildren(this.files, []);
    }

    const iterations = Array.isArray(job.iterations) ? job.iterations : [];
    if (iterations.length) {
      setChildren(this.iterations, [
        el("h2", textWithIcon("steps", "Iterations")),
        el(
          ".submission-iterations-list",
          ...iterations.map((iteration, index) => this.buildIterationCard(job, iteration, index === iterations.length - 1))
        )
      ]);
      this.iterations.style.display = "";
    } else {
      setChildren(this.iterations, [
        el("h2", textWithIcon("steps", "Iterations")),
        el(".empty-state", el("p", "No iteration history is available yet."))
      ]);
      this.iterations.style.display = "";
    }
  }
}

function buildCompanyCard(app, company) {
  const badges = [
    company.isInInvestingUniverse ? createBadge("Investing") : null,
    company.isInTradingUniverse ? createBadge("Trading") : null,
    company.statusActive ? createBadge("Active") : createBadge("Inactive")
  ].filter(Boolean);

  return el(
    "article.job-card",
    el(".job-header", createBadge(company.exchange || "N/A"), el("span.job-date", company.marketCapBucket || "Unknown market cap")),
    el("h3.job-title", el("a.job-link", {
      href: pathForPlatformCompany(company.id),
      onclick: (event) => {
        event.preventDefault();
        app.navigate(pathForPlatformCompany(company.id));
      }
    }, `${company.symbol} • ${company.companyName}`)),
    el(".job-meta", el("span", firstNonEmpty(company.sector, "Unknown sector")), el("span", firstNonEmpty(company.industry, "Unknown industry"))),
    badges.length ? el(".inline-badges", ...badges) : null
  );
}

function buildReviewCard(app, review) {
  return el(
    "article.job-card",
    el(".job-header", createStatusPill(review.reviewStatus), el("span.job-date", formatDateTime(review.reviewDate))),
    el("h3.job-title", el("a.job-link", {
      href: pathForPlatformReview(review.id),
      onclick: (event) => {
        event.preventDefault();
        app.navigate(pathForPlatformReview(review.id));
      }
    }, `${review.symbol} • ${humanizeToken(review.finalActionAfterReview || "pending")}`)),
    el(
      ".job-meta",
      el("span", `Book: ${humanizeToken(review.bookType)}`),
      el("span", `Score: ${formatScore(review.weightedTotalScore)}`),
      el("span", `Confidence: ${formatPercent((review.confidenceScore || 0) * 100)}`)
    ),
    buildInlineBadgeRow([
      review.finalBucketAfterReview ? createBadge(humanizeToken(review.finalBucketAfterReview)) : null
    ])
  );
}

function buildWorkflowRunCard(app, run) {
  return el(
    "article.job-card",
    el(".job-header", createStatusPill(run.status), el("span.job-date", formatDateTime(run.startedAt))),
    el("h3.job-title", el("a.job-link", {
      href: pathForPlatformWorkflowRun(run.id),
      onclick: (event) => {
        event.preventDefault();
        app.navigate(pathForPlatformWorkflowRun(run.id));
      }
    }, `${humanizeToken(run.bookType)} • ${humanizeToken(run.runType)}`)),
    el(
      ".job-meta",
      el("span", `Mode: ${humanizeToken(run.mode || "n/a")}`),
      el("span", `Companies: ${formatNumber(run.companiesScannedCount || 0)}`),
      el("span", `Reviews: ${formatNumber(run.reviewsCreatedCount || 0)}`),
      el("span", `Errors: ${formatNumber(run.errorsCount || 0)}`)
    ),
    buildInlineBadgeRow([
      run.dryRun ? createBadge("Dry Run") : null,
      run.configSnapshotId ? createBadge(`Snapshot ${run.configSnapshotId}`) : null
    ])
  );
}

function buildAIBatchJobCard(app, job) {
  return el(
    "article.job-card",
    el(".job-header", createStatusPill(job.status), el("span.job-date", formatDateTime(job.submittedAt || job.createdAt))),
    el("h3.job-title", el("a.job-link", {
      href: pathForPlatformAIBatchJob(job.id),
      onclick: (event) => {
        event.preventDefault();
        app.navigate(pathForPlatformAIBatchJob(job.id));
      }
    }, `${humanizeToken(job.jobType)} • ${job.id}`)),
    el(
      ".job-meta",
      el("span", `Book: ${humanizeToken(job.bookType)}`),
      el("span", `Provider: ${job.providerName || "N/A"}`),
      el("span", `Workflow: ${job.workflowRunId || "N/A"}`),
      el("span", `Retries: ${formatNumber(job.retryCount || 0)}/${formatNumber(job.maxRetryCount || 0)}`)
    ),
    buildInlineBadgeRow([
      job.providerJobHandle ? createBadge("Provider Handle") : null,
      job.localJobHandle ? createBadge("Local Handle") : null,
      job.completedAt ? createBadge("Completed") : null
    ]),
    job.errorSummary ? el("p.card-preview", job.errorSummary) : null
  );
}

function buildCapitalAllocationCard(app, allocation) {
  return el(
    "article.job-card",
    el(".job-header", createBadge(humanizeToken(allocation.bookType)), el("span.job-date", formatDateTime(allocation.allocationDate))),
    el("h3.job-title", el("a.job-link", {
      href: pathForPlatformCapitalAllocation(allocation.id),
      onclick: (event) => {
        event.preventDefault();
        app.navigate(pathForPlatformCapitalAllocation(allocation.id));
      }
    }, `Allocation Run ${allocation.id}`)),
    el(
      ".job-meta",
      el("span", `Workflow: ${allocation.workflowRunId}`),
      el("span", `Allocated: ${formatCurrency(allocation.allocatedCashTotal)}`),
      el("span", `Cash Left: ${formatCurrency(allocation.cashLeftUnallocated)}`)
    )
  );
}

function buildPositionCard(position) {
  return el(
    "article.job-card",
    el(".job-header", createBadge(humanizeToken(position.bookType)), el("span.job-date", formatDateTime(position.updatedAt))),
    el("h3.job-title", `${position.symbol} • ${formatNumber(position.quantity)} shares`),
    el(
      ".job-meta",
      el("span", `Market value: ${formatCurrency(position.marketValue)}`),
      el("span", `Book: ${formatPercent(position.positionPctOfBook)}`),
      el("span", `Portfolio: ${formatPercent(position.positionPctOfTotalPortfolio)}`)
    )
  );
}

function buildOverrideCard(override) {
  return el(
    "article.job-card",
    el(".job-header", createBadge(humanizeToken(override.bookType)), el("span.job-date", formatDateTime(override.overrideDate))),
    el("h3.job-title", `${humanizeToken(override.originalAction)} → ${humanizeToken(override.overriddenAction)}`),
    el(
      ".job-meta",
      el("span", `Review: ${override.reviewId}`),
      el("span", `Company: ${override.companyId}`)
    )
  );
}

function buildEvidenceCard(reference) {
  const title = firstNonEmpty(reference.sourceTitle, reference.excerptOrMetricName, reference.id, "Evidence");
  return el(
    ".result-box",
    el(".detail-list",
      infoRow("Source", humanizeToken(reference.sourceType)),
      infoRow("Title", title),
      infoRow("Date", formatDateTime(reference.sourceDate)),
      infoRow("Direction", humanizeToken(reference.evidenceDirection)),
      infoRow("Period", reference.sourcePeriod || "N/A"),
      infoRow("Value", firstNonEmpty(reference.excerptOrMetricValue, reference.evidenceSummary, "N/A")),
      infoRow("Reference", reference.sourceUrlOrPath || "N/A")
    )
  );
}

function buildReviewSectionCard(section) {
  return el(
    "article.job-card.review-section-card",
    el(
      ".job-header",
      el("h3.job-title", section.sectionName),
      buildInlineBadgeRow([
        createBadge(`Score ${formatScore(section.sectionScoreRaw)}`),
        createBadge(`Weight ${formatPercent(section.sectionWeight)}`),
        section.sectionPassedMinimumCheck ? createBadge("Passed minimum") : createBadge("Failed minimum"),
        section.sectionActionCap ? createBadge(humanizeToken(section.sectionActionCap)) : null
      ])
    ),
    section.sectionSummary ? el("p.card-preview", section.sectionSummary) : null,
    el(
      ".detail-grid",
      el(
        ".detail-block",
        el("h4.section-subheading", "Strengths"),
        listOrEmpty(section.sectionStrengths, "No strengths recorded.")
      ),
      el(
        ".detail-block",
        el("h4.section-subheading", "Weaknesses"),
        listOrEmpty(section.sectionWeaknesses, "No weaknesses recorded.")
      )
    ),
    el(".detail-block", el("h4.section-subheading", "Risks"), listOrEmpty(section.sectionRisks, "No risks recorded.")),
    el(
      ".detail-block",
      el("h4.section-subheading", "Sub-scores"),
      safeArray(section.subScores).length
        ? el(
            ".subscore-grid",
            ...section.subScores.map((subScore) =>
              el(
                ".metric-card",
                el("p.metric-label", subScore.subScoreName),
                el("strong.metric-value", formatScore(subScore.subScoreValue)),
                el("p.metric-note", `${formatPercent(subScore.subScoreWeight)} • ${humanizeToken(subScore.trendDirection)}`),
                subScore.subScoreSummary ? el("p.metric-footnote", subScore.subScoreSummary) : null
              )
            )
          )
        : el(".empty-state", el("p", "No sub-score detail available."))
    )
  );
}

function buildWorkflowStepCard(step) {
  const disclosure = el("details.execution-step-card");
  disclosure.open = step.status !== "completed";

  const asyncTask = step.asyncTask;
  const summaryMeta = [];
  if (step.durationMs) {
    summaryMeta.push(formatDurationMs(step.durationMs));
  }
  if (asyncTask?.status) {
    summaryMeta.push(`Async ${humanizeToken(asyncTask.status)}`);
  }

  setChildren(disclosure, [
    el(
      "summary.execution-step-summary",
      el(
        ".execution-step-summary-copy",
        el("span.execution-step-number", step.stepName),
        el("span.execution-step-prompt", humanizeToken(step.status)),
        el("span.execution-step-meta", summaryMeta.join(" • ") || "No timing metadata")
      ),
      createStatusPill(step.status)
    ),
    el(
      ".execution-step-content",
      el(
        ".detail-list",
        infoRow("Started", formatDateTime(step.startedAt)),
        infoRow("Completed", formatDateTime(step.completedAt)),
        infoRow("Duration", formatDurationMs(step.durationMs)),
        infoRow("Async Result", formatBool(asyncTask?.resultAvailable))
      ),
      asyncTask
        ? el(
            ".detail-block",
            el("h3", textWithIcon("workflow", "Async Task")),
            el(
              ".detail-list",
              infoRow("Provider", asyncTask.provider || "N/A"),
              infoRow("Task Kind", asyncTask.taskKind || "N/A"),
              infoRow("Submission", asyncTask.submissionId || "N/A"),
              infoRow("Batch", asyncTask.batchId || "N/A"),
              infoRow("Representative Job", asyncTask.representativeJobId || "N/A")
            )
          )
        : null,
      buildJSONSection("input", "Input Snapshot", step.inputSnapshot, "No step input snapshot available."),
      buildJSONSection("output", "Output Snapshot", step.outputSnapshot, "No step output snapshot available."),
      step.error ? buildJSONSection("error", "Step Error", step.error, "No error payload.") : null
    )
  ]);

  return disclosure;
}

function buildWorkflowStepRunCard(step) {
  const disclosure = el("details.execution-step-card");
  disclosure.open = step.status !== "completed";

  const summaryMeta = [];
  if (step.startedAt) {
    summaryMeta.push(`Started ${formatDateTime(step.startedAt)}`);
  }
  if (step.completedAt) {
    summaryMeta.push(`Completed ${formatDateTime(step.completedAt)}`);
  }

  setChildren(disclosure, [
    el(
      "summary.execution-step-summary",
      el(
        ".execution-step-summary-copy",
        el("span.execution-step-number", step.stepName),
        el("span.execution-step-prompt", humanizeToken(step.status)),
        el("span.execution-step-meta", summaryMeta.join(" • ") || "No timing metadata")
      ),
      createStatusPill(step.status)
    ),
    el(
      ".execution-step-content",
      el(
        ".detail-list",
        infoRow("Started", formatDateTime(step.startedAt)),
        infoRow("Completed", formatDateTime(step.completedAt)),
        infoRow("Error Summary", step.errorSummary || "N/A")
      ),
      buildJSONSection("details", "Step Metadata", step.metadata, "No step metadata available.")
    )
  ]);

  return disclosure;
}

function buildAIBatchItemCard(app, item) {
  const actions = [];
  const canRetry = ["failed", "invalid_output"].includes(item.status) || item.validationStatus === "invalid";
  const canSkip = !["completed", "skipped"].includes(item.status);

  if (canRetry) {
    actions.push(
      createActionButton({
        label: "Retry Item",
        iconName: "refresh",
        small: true,
        onclick: async (event) => {
          await app.retryPlatformAIBatchItem(item.id, item.aiBatchJobId, event.currentTarget);
        }
      })
    );
  }

  if (canSkip) {
    actions.push(
      createActionButton({
        label: "Skip Item",
        iconName: "close",
        small: true,
        onclick: async (event) => {
          await app.skipPlatformAIBatchItem(item.id, item.aiBatchJobId, event.currentTarget);
        }
      })
    );
  }

  if (item.targetReviewId) {
    actions.push(createNavigateButton(app, pathForPlatformReview(item.targetReviewId), "Open Review", "open", { small: true }));
  }

  return el(
    "article.job-card",
    el(".job-header", createStatusPill(item.status), el("span.job-date", formatDateTime(item.completedAt || item.updatedAt))),
    el("h3.job-title", `${firstNonEmpty(item.symbol, item.companyId, item.id)} • ${humanizeToken(item.itemType)}`),
    el(
      ".job-meta",
      el("span", `Validation: ${humanizeToken(item.validationStatus)}`),
      el("span", `Workflow: ${item.workflowRunId || "N/A"}`),
      el("span", `Review: ${item.targetReviewId || "N/A"}`),
      el("span", `Thesis: ${item.targetThesisId || "N/A"}`)
    ),
    buildInlineBadgeRow([
      item.errorSummary ? createBadge("Error") : null,
      safeArray(item.validationErrors).length ? createBadge("Validation Errors") : null
    ]),
    item.errorSummary ? el("p.card-preview", item.errorSummary) : null,
    actions.length ? el(".top-actions", ...actions) : null,
    buildJSONSection("input", "Input Snapshot", item.inputPayload, "No item input payload available."),
    buildJSONSection("output", "Result Payload", item.resultPayload, "No result payload is stored yet."),
    safeArray(item.validationErrors).length
      ? el(".detail-block", el("h3", textWithIcon("error", "Validation Errors")), listOrEmpty(item.validationErrors, "No validation errors."))
      : null
  );
}

class UnavailablePage {
  constructor(app) {
    this.app = app;
    this.title = el("h1.detail-title", "Workspace unavailable");
    this.copy = el("p.hero-text");
    this.actions = el(".top-actions");
    this.el = el(".panel.detail-panel", this.title, this.copy, this.actions);
  }

  update(kind) {
    const isPlatform = kind === "platform";
    this.title.textContent = isPlatform ? "Platform routes are not available here." : "Batch routes are not available here.";
    this.copy.textContent = isPlatform
      ? "This server is not exposing the investing platform API. Switch to the platform backend to inspect companies, reviews, workflow runs, and allocations."
      : "This server is not exposing the original batch automation API. Use the platform routes from the navigation to inspect the investing system.";
    setChildren(
      this.actions,
      isPlatform
        ? []
        : [createNavigateButton(this.app, "/platform", "Open Platform", "platform", { variant: "primary" })]
    );
  }
}

class PlatformHomePage {
  constructor(app) {
    this.app = app;
    this.metrics = el(".metrics-grid");
    this.actions = el(".top-actions");
    this.summary = el(".detail-grid");
    this.el = el(
      ".panel.detail-panel",
      el("p.eyebrow", textWithIcon("platform", "Platform")),
      el("h1.detail-title", "Inspect the investing system foundation."),
      el(
        "p.hero-text",
        "This UI exposes the persisted company, review, thesis, workflow, allocation, config, and override records so future AI layers stay auditable and explainable."
      ),
      this.metrics,
      this.actions,
      this.summary
    );
  }

  update(state) {
    const config = state.currentConfig || {};
    const globalConfig = config.global || {};
    const investingConfig = config.investing || {};
    const portfolioSplit = investingConfig.allocation?.portfolioTargetSplit || {};

    setChildren(this.metrics, [
      buildMetricCard("Companies", formatNumber(safeArray(state.platformCompanies).length, "0"), "Loaded into the current view"),
      buildMetricCard("Reviews", formatNumber(safeArray(state.platformReviews).length, "0"), "Recent review snapshots"),
      buildMetricCard("Workflow Runs", formatNumber(safeArray(state.platformWorkflowRuns).length, "0"), "Persisted investing or trading runs"),
      buildMetricCard("AI Batch Jobs", formatNumber(safeArray(state.platformAIBatchJobs).length, "0"), "Async provider submissions"),
      buildMetricCard("Positions", formatNumber(safeArray(state.platformPositions).length, "0"), "Materialized current positions")
    ]);

    setChildren(this.actions, [
      createNavigateButton(this.app, "/platform/companies", "Browse Companies", "company", { variant: "primary" }),
      createNavigateButton(this.app, "/platform/reviews", "Review Archive", "review"),
      createNavigateButton(this.app, "/platform/workflow-runs", "Workflow Runs", "workflow"),
      createNavigateButton(this.app, "/platform/ai-batch-jobs", "AI Batch Jobs", "aijob"),
      createNavigateButton(this.app, "/platform/config", "Inspect Config", "config")
    ]);

    setChildren(this.summary, [
      el(
        ".detail-block",
        el("h2", textWithIcon("details", "Runtime")),
        el(
          ".detail-list",
          infoRow("Environment", config.environment || "N/A"),
          infoRow("Timezone", globalConfig.defaultTimezone || "N/A"),
          infoRow("Investing Mode", humanizeToken(investingConfig.defaultMode)),
          infoRow("Async AI Enabled", formatBool(config.asyncAi?.enabled))
        )
      ),
      el(
        ".detail-block",
        el("h2", textWithIcon("capital", "Portfolio Split")),
        el(
          ".detail-list",
          infoRow("Investing Book", formatPercent(portfolioSplit.investingBookPct)),
          infoRow("Trading Book", formatPercent(portfolioSplit.tradingBookPct)),
          infoRow("Liquid Reserve", formatPercent(portfolioSplit.liquidReservePct)),
          infoRow("Default Tranches", formatNumber(investingConfig.allocation?.defaultTrancheCount))
        )
      )
    ]);
  }
}

class PlatformCompaniesPage {
  constructor(app) {
    this.app = app;
    this.count = el("p");
    this.error = el("p.form-error");
    this.searchInput = el("input.input-control", {
      type: "search",
      placeholder: "Search by company name or symbol"
    });
    this.searchButton = createActionButton({
      label: "Search",
      iconName: "search",
      variant: "primary",
      onclick: async () => {
        try {
          this.setError("");
          await this.app.loadPlatformCompanies(this.searchInput.value.trim());
        } catch (error) {
          this.setError(error.message);
          this.app.setMessage(error.message);
        }
      }
    });
    this.cardsHost = el(".job-list");
    this.empty = el(".empty-state", el("p", "No companies matched this query."));
    this.el = el(
      ".panel.detail-panel",
      el(
        ".section-heading",
        el("h2", textWithIcon("company", "Companies")),
        el(".top-actions", this.count, createNavigateButton(this.app, "/platform/workflow-runs", "Start Run", "workflow"))
      ),
      el(".compact-field-grid", el(".field-block", el("label.label", "Search"), this.searchInput), this.searchButton),
      this.error,
      this.empty,
      this.cardsHost
    );
    this.setError("");
  }

  setBusy(isBusy) {
    this.searchInput.disabled = isBusy;
    this.searchButton.disabled = isBusy;
  }

  setError(message) {
    this.error.textContent = message || "";
    this.error.style.display = message ? "" : "none";
  }

  update(companies, search = "") {
    this.searchInput.value = search || "";
    this.count.textContent = `${companies.length} loaded`;
    this.empty.style.display = companies.length ? "none" : "";
    setChildren(this.cardsHost, companies.map((company) => buildCompanyCard(this.app, company)));
  }
}

class PlatformCompanyDetailPage {
  constructor(app) {
    this.app = app;
    this.topline = el(".detail-topline");
    this.title = el("h1.detail-title");
    this.overview = el(".detail-grid");
    this.history = el(".detail-block");
    this.thesis = el(".detail-block");
    this.reviews = el(".detail-block");
    this.raw = el(".detail-block");
    this.el = el(".panel.detail-panel", this.topline, this.title, this.overview, this.history, this.thesis, this.reviews, this.raw);
  }

  update(company, reviews, thesis, historySummary) {
    const backButton = createNavigateButton(this.app, "/platform/companies", "Back to Companies", "back", { small: true });
    setChildren(this.topline, [
      backButton,
      buildInlineBadgeRow([
        company.isInInvestingUniverse ? createBadge("Investing") : null,
        company.isInTradingUniverse ? createBadge("Trading") : null,
        company.statusActive ? createBadge("Active") : createBadge("Inactive")
      ])
    ]);
    this.title.textContent = `${company.symbol} • ${company.companyName}`;

    setChildren(this.overview, [
      el(
        ".detail-block",
        el("h2", textWithIcon("company", "Company")),
        el(
          ".detail-list",
          infoRow("Exchange", company.exchange || "N/A"),
          infoRow("Sector", company.sector || "N/A"),
          infoRow("Industry", company.industry || "N/A"),
          infoRow("Sub-industry", company.subIndustry || "N/A"),
          infoRow("Listing Date", formatDateTime(company.listingDate)),
          infoRow("Market Cap Bucket", company.marketCapBucket || "N/A")
        ),
        company.businessSummary ? el("p.card-preview", company.businessSummary) : null
      ),
      el(
        ".detail-block",
        el("h2", textWithIcon("history", "History Summary")),
        historySummary
          ? el(
              ".detail-list",
              infoRow("Review Count", formatNumber(historySummary.reviewCount || 0)),
              infoRow("Latest Score", formatScore(historySummary.latestScore)),
              infoRow("Latest Action", humanizeToken(historySummary.latestAction)),
              infoRow("Latest Bucket", humanizeToken(historySummary.latestBucket)),
              infoRow("Has Thesis", formatBool(historySummary.hasThesis)),
              infoRow("Currently Owned", formatBool(historySummary.isOwned))
            )
          : el(".empty-state", el("p", "No history summary available yet."))
      )
    ]);

    setChildren(this.history, [
      el("h2", textWithIcon("history", "Raw History Snapshot")),
      resultJsonBlock(historySummary, "No history snapshot is available.")
    ]);

    if (thesis) {
      setChildren(this.thesis, [
        el("h2", textWithIcon("thesis", "Active Thesis")),
        el(
          ".detail-grid",
          el(
            ".detail-block",
            el(".detail-list",
              infoRow("Status", humanizeToken(thesis.thesisStatus)),
              infoRow("Version", formatNumber(thesis.thesisVersion)),
              infoRow("Health Score", formatScore(thesis.thesisHealthScore)),
              infoRow("Confidence", formatPercent((thesis.confidenceLevel || 0) * 100))
            ),
            el("p.card-preview", thesis.thesisSummary)
          ),
          el(
            ".detail-block",
            el("h3.section-subheading", "Why It Can Compound"),
            el("p.card-preview", thesis.whyThisBusinessCanCompound)
          )
        ),
        el(".detail-grid",
          el(".detail-block", el("h3.section-subheading", "Growth Drivers"), listOrEmpty(thesis.keyGrowthDrivers, "No growth drivers recorded.")),
          el(".detail-block", el("h3.section-subheading", "Key Risks"), listOrEmpty(thesis.keyRisks, "No risks recorded."))
        )
      ]);
      this.thesis.style.display = "";
    } else {
      this.thesis.style.display = "";
      setChildren(this.thesis, [el("h2", textWithIcon("thesis", "Active Thesis")), el(".empty-state", el("p", "No active thesis exists for this company yet."))]);
    }

    setChildren(this.reviews, [
      el(".section-heading", el("h2", textWithIcon("review", "Recent Reviews")), el("p", `${reviews.length} loaded`)),
      reviews.length
        ? el(".job-list", ...reviews.map((review) => buildReviewCard(this.app, review)))
        : el(".empty-state", el("p", "No review history exists for this company yet."))
    ]);

    setChildren(this.raw, [
      el("h2", textWithIcon("raw", "Raw Company JSON")),
      resultJsonBlock(company, "Company record is unavailable.")
    ]);
  }
}

class PlatformReviewsPage {
  constructor(app) {
    this.app = app;
    this.count = el("p");
    this.filter = el("select.select-control", {
      onchange: async () => {
        try {
          await this.app.loadPlatformReviews(this.filter.value);
        } catch (error) {
          this.app.setMessage(error.message);
        }
      }
    },
    el("option", { value: "" }, "All Books"),
    el("option", { value: "investing" }, "Investing"),
    el("option", { value: "trading" }, "Trading"));
    this.cardsHost = el(".job-list");
    this.empty = el(".empty-state", el("p", "No reviews are available yet."));
    this.el = el(
      ".panel.detail-panel",
      el(".section-heading", el("h2", textWithIcon("review", "Reviews")), el(".top-actions", this.count)),
      el(".field-block", el("label.label", "Filter by book"), this.filter),
      this.empty,
      this.cardsHost
    );
  }

  update(reviews, bookType = "") {
    this.filter.value = bookType || "";
    this.count.textContent = `${reviews.length} loaded`;
    this.empty.style.display = reviews.length ? "none" : "";
    setChildren(this.cardsHost, reviews.map((review) => buildReviewCard(this.app, review)));
  }
}

class PlatformReviewDetailPage {
  constructor(app) {
    this.app = app;
    this.topline = el(".detail-topline");
    this.title = el("h1.detail-title");
    this.meta = el(".detail-grid");
    this.decision = el(".detail-block");
    this.changeLog = el(".detail-block");
    this.sections = el(".detail-block");
    this.evidence = el(".detail-block");
    this.raw = el(".detail-block");
    this.el = el(".panel.detail-panel", this.topline, this.title, this.meta, this.decision, this.changeLog, this.sections, this.evidence, this.raw);
  }

  update(review, diff, evidence) {
    const backButton = createNavigateButton(this.app, "/platform/reviews", "Back to Reviews", "back", { small: true });
    const overrideButton = createActionButton({
      label: "Create Override",
      iconName: "override",
      small: true,
      onclick: () => this.app.prepareOverrideFromReview(review)
    });
    setChildren(this.topline, [backButton, createStatusPill(review.reviewStatus), overrideButton]);
    this.title.textContent = `${review.symbol} • ${humanizeToken(review.finalActionAfterReview || "pending review")}`;

    setChildren(this.meta, [
      el(
        ".detail-block",
        el("h2", textWithIcon("score", "Review Snapshot")),
        el(
          ".detail-list",
          infoRow("Book", humanizeToken(review.bookType)),
          infoRow("Review Date", formatDateTime(review.reviewDate)),
          infoRow("Period", humanizeToken(review.reviewPeriodType)),
          infoRow("Weighted Score", formatScore(review.weightedTotalScore)),
          infoRow("Confidence", formatPercent((review.confidenceScore || 0) * 100)),
          infoRow("Mode", humanizeToken(review.mode))
        )
      ),
      el(
        ".detail-block",
        el("h2", textWithIcon("bucket", "Outcome")),
        el(
          ".detail-list",
          infoRow("Action", humanizeToken(review.finalActionAfterReview)),
          infoRow("Bucket", humanizeToken(review.finalBucketAfterReview)),
          infoRow("Hard Gate Failed", formatBool(review.hardGateFailed)),
          infoRow("Reviewer", humanizeToken(review.reviewerType)),
          infoRow("AI Model", review.aiModelName || "N/A"),
          infoRow("Prompt Version", review.aiPromptVersion || "N/A")
        )
      )
    ]);

    setChildren(this.decision, [
      el("h2", textWithIcon("details", "Decision")),
      review.decisionAction
        ? el(
            ".detail-grid",
            el(
              ".detail-block",
              el(
                ".detail-list",
                infoRow("Primary Reason", review.decisionAction.actionReasonPrimary || "N/A"),
                infoRow("Secondary Reason", review.decisionAction.actionReasonSecondary || "N/A"),
                infoRow("Capital Eligible", formatBool(review.decisionAction.capitalEligible)),
                infoRow("Capital Priority", formatScore(review.decisionAction.capitalPriorityScore)),
                infoRow("Target Position", formatPercent(review.decisionAction.recommendedPositionTargetPct)),
                infoRow("Position Cap", formatPercent(review.decisionAction.recommendedPositionCapPct))
              )
            ),
            el(
              ".detail-block",
              el("h3.section-subheading", "Constraints"),
              listOrEmpty(review.decisionAction.actionConstraints, "No action constraints were recorded."),
              review.actionRationaleSummary ? el("p.card-preview", review.actionRationaleSummary) : null
            )
          )
        : el(".empty-state", el("p", "No mapped decision action is stored on this review."))
    ]);

    setChildren(this.changeLog, [
      el("h2", textWithIcon("history", "Change Log")),
      diff
        ? el(
            ".detail-grid",
            el(
              ".detail-block",
              el(
                ".detail-list",
                infoRow("Previous Review", diff.previousReviewId || "N/A"),
                infoRow("Weighted Score Change", formatScore(diff.weightedTotalScoreChange)),
                infoRow("Bucket Change", diff.bucketChange || "N/A"),
                infoRow("Action Change", diff.actionChange || "N/A"),
                infoRow("Exit Review Required", formatBool(diff.requiresExitReview))
              )
            ),
            el(
              ".detail-block",
              el("h3.section-subheading", "Summary"),
              el("p.card-preview", diff.changeSummary || "No change summary provided.")
            )
          )
        : el(".empty-state", el("p", "No change log exists yet."))
    ]);

    setChildren(this.sections, [
      el(".section-heading", el("h2", textWithIcon("score", "Sections")), el("p", `${safeArray(review.sections).length} sections`)),
      safeArray(review.sections).length
        ? el(".job-list", ...review.sections.map((section) => buildReviewSectionCard(section)))
        : el(".empty-state", el("p", "No section scorecards are stored for this review."))
    ]);

    setChildren(this.evidence, [
      el(".section-heading", el("h2", textWithIcon("files", "Evidence")), el("p", `${safeArray(evidence).length} references`)),
      safeArray(evidence).length
        ? el(".evidence-grid", ...evidence.map((reference) => buildEvidenceCard(reference)))
        : el(".empty-state", el("p", "No evidence references are available for this review."))
    ]);

    setChildren(this.raw, [el("h2", textWithIcon("raw", "Raw Review JSON")), resultJsonBlock(review, "Review payload unavailable.")]);
  }
}

class InvestingWorkflowStartPanel {
  constructor(app) {
    this.app = app;
    this.error = el("p.form-error");
    this.runType = el("select.select-control",
      el("option", { value: "manual" }, "Manual"),
      el("option", { value: "monthly_scan" }, "Monthly Scan"),
      el("option", { value: "quarterly_refresh" }, "Quarterly Refresh"),
      el("option", { value: "event_refresh" }, "Event Refresh")
    );
    this.mode = el("select.select-control",
      el("option", { value: "balanced" }, "Balanced"),
      el("option", { value: "early_hunter" }, "Early Hunter"),
      el("option", { value: "confirmed_compounder" }, "Confirmed Compounder")
    );
    this.limit = el("input.input-control", { type: "number", min: "1", value: "10" });
    this.requestedBy = el("input.input-control", { type: "text", placeholder: "local-dev" });
    this.idempotencyKey = el("input.input-control", { type: "text", placeholder: "optional-manual-key" });
    this.notes = el("textarea", { rows: 4, placeholder: "Optional notes for this run." });
    this.dryRunButton = createActionButton({
      label: "Dry Run",
      iconName: "workflow",
      onclick: async (event) => {
        await this.submit(true, event.currentTarget);
      }
    });
    this.startButton = createActionButton({
      label: "Start Async Run",
      iconName: "start",
      variant: "primary",
      onclick: async (event) => {
        await this.submit(false, event.currentTarget);
      }
    });
    this.el = el(
      ".panel.detail-panel",
      el(".section-heading", el("h2", textWithIcon("workflow", "Start Investing Workflow")), el("p", "Async-only batch submission for review generation.")),
      el(
        ".detail-grid",
        el(".field-block", el("label.label", "Run Type"), this.runType),
        el(".field-block", el("label.label", "Mode"), this.mode),
        el(".field-block", el("label.label", "Company Limit"), this.limit),
        el(".field-block", el("label.label", "Requested By"), this.requestedBy)
      ),
      el(".field-block", el("label.label", "Idempotency Key"), this.idempotencyKey),
      el(".field-block", el("label.label", "Notes"), this.notes),
      this.error,
      el(".prompt-actions", this.dryRunButton, this.startButton)
    );
    this.setError("");
  }

  setBusy(isBusy) {
    [this.runType, this.mode, this.limit, this.requestedBy, this.idempotencyKey, this.notes, this.dryRunButton, this.startButton].forEach((field) => {
      field.disabled = isBusy;
    });
  }

  setError(message) {
    this.error.textContent = message || "";
    this.error.style.display = message ? "" : "none";
  }

  getPayload() {
    const limit = Number.parseInt(this.limit.value || "0", 10);
    return {
      runType: this.runType.value,
      mode: this.mode.value,
      limit: Number.isNaN(limit) || limit <= 0 ? 10 : limit,
      requestedBy: this.requestedBy.value.trim(),
      idempotencyKey: this.idempotencyKey.value.trim(),
      notes: this.notes.value.trim()
    };
  }

  async submit(dryRun, button) {
    try {
      this.setError("");
      await this.app.startPlatformInvestingWorkflow(this.getPayload(), dryRun, button);
    } catch (error) {
      this.setError(error.message);
    }
  }
}

class PlatformWorkflowRunsPage {
  constructor(app) {
    this.app = app;
    this.count = el("p");
    this.startPanel = new InvestingWorkflowStartPanel(app);
    this.cardsHost = el(".job-list");
    this.empty = el(".empty-state", el("p", "No workflow runs are stored yet."));
    this.el = el(
      ".detail-panel",
      this.startPanel.el,
      el(
        ".panel.detail-panel",
        el(".section-heading", el("h2", textWithIcon("workflow", "Workflow Runs")), el(".top-actions", this.count)),
        this.empty,
        this.cardsHost
      )
    );
  }

  setBusy(isBusy) {
    this.startPanel.setBusy(isBusy);
  }

  update(runs) {
    this.count.textContent = `${runs.length} loaded`;
    this.empty.style.display = runs.length ? "none" : "";
    setChildren(this.cardsHost, runs.map((run) => buildWorkflowRunCard(this.app, run)));
  }
}

class PlatformWorkflowRunDetailPage {
  constructor(app) {
    this.app = app;
    this.topline = el(".detail-topline");
    this.title = el("h1.detail-title");
    this.summary = el(".detail-grid");
    this.batchJobs = el(".detail-block");
    this.steps = el(".detail-block");
    this.status = el(".detail-block");
    this.raw = el(".detail-block");
    this.el = el(".panel.detail-panel", this.topline, this.title, this.summary, this.batchJobs, this.steps, this.status, this.raw);
  }

  update(run, summary, status, steps, batchJobs) {
    const actions = [
      createNavigateButton(this.app, "/platform/workflow-runs", "Back to Runs", "back", { small: true }),
      createStatusPill(run.status),
      createActionButton({
        label: "Resume",
        iconName: "start",
        small: true,
        onclick: async (event) => {
          await this.app.resumePlatformWorkflowRun(run.id, event.currentTarget);
        }
      }),
      createActionButton({
        label: "Reconcile",
        iconName: "refresh",
        small: true,
        onclick: async (event) => {
          await this.app.reconcilePlatformWorkflowRun(run.id, event.currentTarget);
        }
      })
    ];

    setChildren(this.topline, [
      ...actions
    ]);
    this.title.textContent = `${humanizeToken(run.bookType)} • ${humanizeToken(run.runType)}`;

    setChildren(this.summary, [
      el(
        ".detail-block",
        el("h2", textWithIcon("details", "Run Summary")),
        el(
          ".detail-list",
          infoRow("Mode", humanizeToken(run.mode)),
          infoRow("Dry Run", formatBool(run.dryRun)),
          infoRow("Started", formatDateTime(run.startedAt)),
          infoRow("Completed", formatDateTime(run.completedAt)),
          infoRow("Config Snapshot", run.configSnapshotId || "N/A"),
          infoRow("Idempotency Key", run.idempotencyKey || "N/A"),
          infoRow("Run ID", run.id || "N/A")
        )
      ),
      el(
        ".detail-block",
        el("h2", textWithIcon("score", "Counts")),
        el(
          ".detail-list",
          infoRow("Companies Scanned", formatNumber(summary?.companiesScannedCount ?? run.companiesScannedCount ?? 0)),
          infoRow("Reviews Created", formatNumber(summary?.reviewsCreatedCount ?? run.reviewsCreatedCount ?? 0)),
          infoRow("Errors", formatNumber(summary?.errorsCount ?? run.errorsCount ?? 0)),
          infoRow("Completed Steps", formatNumber(summary?.completedSteps ?? 0)),
          infoRow("Waiting Steps", formatNumber(summary?.waitingSteps ?? 0)),
          infoRow("Failed Steps", formatNumber(summary?.failedSteps ?? 0)),
          infoRow("Batch Jobs", formatNumber(safeArray(batchJobs).length))
        )
      )
    ]);

    setChildren(this.batchJobs, [
      el(".section-heading", el("h2", textWithIcon("aijob", "Related Batch Jobs")), el("p", `${safeArray(batchJobs).length} jobs`)),
      safeArray(batchJobs).length
        ? el(".job-list", ...batchJobs.map((job) => buildAIBatchJobCard(this.app, job)))
        : el(".empty-state", el("p", "No AI batch jobs are linked to this workflow run yet."))
    ]);

    setChildren(this.steps, [
      el(".section-heading", el("h2", textWithIcon("steps", "Workflow Steps")), el("p", `${safeArray(steps).length} persisted steps`)),
      safeArray(steps).length
        ? el(".execution-steps-list", ...steps.map((step) => buildWorkflowStepRunCard(step)))
        : el(".empty-state", el("p", "No step history is stored for this run."))
    ]);

    setChildren(this.status, [
      el("h2", textWithIcon("workflow", "Live Status Snapshot")),
      resultJsonBlock(status, "No workflow status payload is available.")
    ]);

    setChildren(this.raw, [
      el("h2", textWithIcon("raw", "Raw Workflow JSON")),
      resultJsonBlock(run, "Workflow run payload unavailable."),
      safeArray(run.stepStatuses).length
        ? el(".detail-block", el("h3", textWithIcon("steps", "Embedded Step Statuses")), resultJsonBlock(run.stepStatuses, "No embedded step statuses."))
        : null
    ]);
  }
}

class PlatformAIBatchJobsPage {
  constructor(app) {
    this.app = app;
    this.count = el("p");
    this.cardsHost = el(".job-list");
    this.empty = el(".empty-state", el("p", "No AI batch jobs have been created yet."));
    this.el = el(
      ".panel.detail-panel",
      el(".section-heading", el("h2", textWithIcon("aijob", "AI Batch Jobs")), el(".top-actions", this.count)),
      this.empty,
      this.cardsHost
    );
  }

  update(jobs) {
    this.count.textContent = `${jobs.length} loaded`;
    this.empty.style.display = jobs.length ? "none" : "";
    setChildren(this.cardsHost, jobs.map((job) => buildAIBatchJobCard(this.app, job)));
  }
}

class PlatformAIBatchJobDetailPage {
  constructor(app) {
    this.app = app;
    this.topline = el(".detail-topline");
    this.title = el("h1.detail-title");
    this.summary = el(".detail-grid");
    this.items = el(".detail-block");
    this.raw = el(".detail-block");
    this.el = el(".panel.detail-panel", this.topline, this.title, this.summary, this.items, this.raw);
  }

  update(job, items) {
    const completedCount = safeArray(items).filter((item) => item.status === "completed").length;
    const failedCount = safeArray(items).filter((item) => ["failed", "invalid_output"].includes(item.status)).length;
    const invalidCount = safeArray(items).filter((item) => item.validationStatus === "invalid").length;

    setChildren(this.topline, [
      createNavigateButton(this.app, "/platform/ai-batch-jobs", "Back to AI Batch Jobs", "back", { small: true }),
      createStatusPill(job.status),
      createActionButton({
        label: "Retry Job",
        iconName: "refresh",
        small: true,
        onclick: async (event) => {
          await this.app.retryPlatformAIBatchJob(job.id, event.currentTarget);
        }
      }),
      job.workflowRunId ? createNavigateButton(this.app, pathForPlatformWorkflowRun(job.workflowRunId), "Open Workflow", "open", { small: true }) : null
    ].filter(Boolean));

    this.title.textContent = `${humanizeToken(job.jobType)} • ${job.id}`;

    setChildren(this.summary, [
      el(
        ".detail-block",
        el("h2", textWithIcon("details", "Job Summary")),
        el(
          ".detail-list",
          infoRow("Book", humanizeToken(job.bookType)),
          infoRow("Provider", job.providerName || "N/A"),
          infoRow("Workflow Run", job.workflowRunId || "N/A"),
          infoRow("Provider Handle", job.providerJobHandle || "N/A"),
          infoRow("Local Handle", job.localJobHandle || "N/A"),
          infoRow("Submitted", formatDateTime(job.submittedAt)),
          infoRow("Last Polled", formatDateTime(job.lastPolledAt)),
          infoRow("Completed", formatDateTime(job.completedAt)),
          infoRow("Retries", `${formatNumber(job.retryCount || 0)} / ${formatNumber(job.maxRetryCount || 0)}`)
        )
      ),
      el(
        ".detail-block",
        el("h2", textWithIcon("score", "Item Counts")),
        el(
          ".detail-list",
          infoRow("Total Items", formatNumber(safeArray(items).length)),
          infoRow("Completed", formatNumber(completedCount)),
          infoRow("Failed", formatNumber(failedCount)),
          infoRow("Invalid", formatNumber(invalidCount)),
          infoRow("Idempotency Key", job.idempotencyKey || "N/A")
        ),
        job.errorSummary ? el("p.card-preview", job.errorSummary) : null
      )
    ]);

    setChildren(this.items, [
      el(".section-heading", el("h2", textWithIcon("review", "Batch Items")), el("p", `${safeArray(items).length} items`)),
      safeArray(items).length
        ? el(".job-list", ...items.map((item) => buildAIBatchItemCard(this.app, item)))
        : el(".empty-state", el("p", "No AI batch items are stored for this job yet."))
    ]);

    setChildren(this.raw, [
      el("h2", textWithIcon("raw", "Raw Batch Job JSON")),
      resultJsonBlock(job, "AI batch job payload unavailable.")
    ]);
  }
}

class PlatformCapitalAllocationsPage {
  constructor(app) {
    this.app = app;
    this.count = el("p");
    this.cardsHost = el(".job-list");
    this.empty = el(".empty-state", el("p", "No capital allocation runs are stored yet."));
    this.el = el(
      ".panel.detail-panel",
      el(".section-heading", el("h2", textWithIcon("allocation", "Capital Allocations")), el(".top-actions", this.count)),
      this.empty,
      this.cardsHost
    );
  }

  update(runs) {
    this.count.textContent = `${runs.length} loaded`;
    this.empty.style.display = runs.length ? "none" : "";
    setChildren(this.cardsHost, runs.map((run) => buildCapitalAllocationCard(this.app, run)));
  }
}

class PlatformCapitalAllocationDetailPage {
  constructor(app) {
    this.app = app;
    this.topline = el(".detail-topline");
    this.title = el("h1.detail-title", "Capital Allocation");
    this.summary = el(".detail-grid");
    this.items = el(".detail-block");
    this.raw = el(".detail-block");
    this.el = el(".panel.detail-panel", this.topline, this.title, this.summary, this.items, this.raw);
  }

  update(run) {
    setChildren(this.topline, [createNavigateButton(this.app, "/platform/capital-allocations", "Back to Allocations", "back", { small: true })]);
    this.title.textContent = `Allocation ${run.id}`;
    setChildren(this.summary, [
      el(
        ".detail-block",
        el("h2", textWithIcon("capital", "Cash Summary")),
        el(
          ".detail-list",
          infoRow("Book", humanizeToken(run.bookType)),
          infoRow("Workflow Run", run.workflowRunId || "N/A"),
          infoRow("Allocation Date", formatDateTime(run.allocationDate)),
          infoRow("Available Cash", formatCurrency(run.availableCashStart)),
          infoRow("Target Deployable", formatCurrency(run.targetDeployableCash)),
          infoRow("Allocated", formatCurrency(run.allocatedCashTotal)),
          infoRow("Cash Left", formatCurrency(run.cashLeftUnallocated))
        )
      ),
      el(
        ".detail-block",
        el("h2", textWithIcon("details", "Notes")),
        el("p.card-preview", run.allocationNotes || "No allocation notes recorded.")
      )
    ]);
    setChildren(this.items, [
      el(".section-heading", el("h2", textWithIcon("allocation", "Allocation Items")), el("p", `${safeArray(run.items).length} items`)),
      safeArray(run.items).length
        ? el(
            ".job-list",
            ...run.items.map((item) =>
              el(
                "article.job-card",
                el(".job-header", createBadge(humanizeToken(item.actionType)), item.blockedByConstraint ? createBadge("Blocked") : createBadge("Eligible")),
                el("h3.job-title", `${item.companyId} • ${formatCurrency(item.recommendedAllocationAmount)}`),
                el(
                  ".job-meta",
                  el("span", `Review: ${item.decisionReviewId}`),
                  el("span", `Priority: ${formatScore(item.capitalPriorityScore)}`),
                  el("span", `Run %: ${formatPercent(item.recommendedAllocationPctOfRun)}`)
                ),
                item.allocationReason ? el("p.card-preview", item.allocationReason) : null,
                item.constraintReason ? el("p.card-preview", item.constraintReason) : null
              )
            )
          )
        : el(".empty-state", el("p", "No allocation items are stored on this run."))
    ]);
    setChildren(this.raw, [el("h2", textWithIcon("raw", "Raw Allocation JSON")), resultJsonBlock(run, "Allocation payload unavailable.")]);
  }
}

class PlatformPositionsPage {
  constructor(app) {
    this.app = app;
    this.count = el("p");
    this.filterButtons = el(".top-actions");
    this.cardsHost = el(".job-list");
    this.empty = el(".empty-state", el("p", "No positions are available in the current projection."));
    this.el = el(
      ".panel.detail-panel",
      el(".section-heading", el("h2", textWithIcon("portfolio", "Positions")), el(".top-actions", this.count)),
      this.filterButtons,
      this.empty,
      this.cardsHost
    );
  }

  update(positions, activeBookType = "") {
    this.count.textContent = `${positions.length} loaded`;
    setChildren(this.filterButtons, [
      createActionButton({
        label: "All Books",
        iconName: "portfolio",
        variant: activeBookType === "" ? "primary" : "secondary",
        onclick: async () => {
          try {
            await this.app.loadPlatformPositions("");
          } catch (error) {
            this.app.setMessage(error.message);
          }
        }
      }),
      createActionButton({
        label: "Investing",
        iconName: "company",
        variant: activeBookType === "investing" ? "primary" : "secondary",
        onclick: async () => {
          try {
            await this.app.loadPlatformPositions("investing");
          } catch (error) {
            this.app.setMessage(error.message);
          }
        }
      }),
      createActionButton({
        label: "Trading",
        iconName: "workflow",
        variant: activeBookType === "trading" ? "primary" : "secondary",
        onclick: async () => {
          try {
            await this.app.loadPlatformPositions("trading");
          } catch (error) {
            this.app.setMessage(error.message);
          }
        }
      })
    ]);
    this.empty.style.display = positions.length ? "none" : "";
    setChildren(this.cardsHost, positions.map((position) => buildPositionCard(position)));
  }
}

class PlatformConfigPage {
  constructor(app) {
    this.app = app;
    this.summary = el(".detail-grid");
    this.current = el(".detail-block");
    this.snapshots = el(".detail-block");
    this.el = el(
      ".panel.detail-panel",
      el(".section-heading", el("h2", textWithIcon("config", "Configuration")), el(".top-actions")),
      this.summary,
      this.current,
      this.snapshots
    );
  }

  update(config, snapshots) {
    const globalConfig = config?.global || {};
    const asyncConfig = config?.asyncAi || {};
    const investingConfig = config?.investing || {};

    setChildren(this.summary, [
      el(
        ".detail-block",
        el("h2", textWithIcon("details", "Current Summary")),
        el(
          ".detail-list",
          infoRow("Environment", config?.environment || "N/A"),
          infoRow("Schema Version", config?.schemaVersion || "N/A"),
          infoRow("Timezone", globalConfig.defaultTimezone || "N/A"),
          infoRow("Default Investing Mode", humanizeToken(investingConfig.defaultMode)),
          infoRow("Async Provider", asyncConfig.provider || "N/A"),
          infoRow("Async Model", asyncConfig.model || "N/A")
        )
      ),
      el(
        ".detail-block",
        el("h2", textWithIcon("capital", "Feature Flags")),
        resultJsonBlock(globalConfig.featureFlags, "No feature flags available.")
      )
    ]);

    setChildren(this.current, [el("h2", textWithIcon("raw", "Current Config JSON")), resultJsonBlock(config, "Current config unavailable.")]);

    setChildren(this.snapshots, [
      el(".section-heading", el("h2", textWithIcon("history", "Config Snapshots")), el("p", `${snapshots.length} loaded`)),
      snapshots.length
        ? el(
            ".job-list",
            ...snapshots.map((snapshot) =>
              el(
                "article.job-card",
                el(".job-header", createBadge(humanizeToken(snapshot.bookType)), el("span.job-date", formatDateTime(snapshot.createdAt))),
                el("h3.job-title", el("a.job-link", {
                  href: pathForPlatformConfigSnapshot(snapshot.id),
                  onclick: (event) => {
                    event.preventDefault();
                    this.app.navigate(pathForPlatformConfigSnapshot(snapshot.id));
                  }
                }, `${humanizeToken(snapshot.mode)} • ${snapshot.id}`)),
                el(".job-meta", el("span", `Schema: ${snapshot.schemaVersion}`))
              )
            )
          )
        : el(".empty-state", el("p", "No config snapshots have been persisted yet."))
    ]);
  }
}

class PlatformConfigSnapshotDetailPage {
  constructor(app) {
    this.app = app;
    this.topline = el(".detail-topline");
    this.title = el("h1.detail-title", "Config Snapshot");
    this.summary = el(".detail-grid");
    this.raw = el(".detail-block");
    this.el = el(".panel.detail-panel", this.topline, this.title, this.summary, this.raw);
  }

  update(snapshot) {
    setChildren(this.topline, [createNavigateButton(this.app, "/platform/config", "Back to Config", "back", { small: true })]);
    this.title.textContent = `${humanizeToken(snapshot.bookType)} • ${humanizeToken(snapshot.mode)}`;
    setChildren(this.summary, [
      el(
        ".detail-block",
        el("h2", textWithIcon("details", "Snapshot Details")),
        el(
          ".detail-list",
          infoRow("Snapshot ID", snapshot.id),
          infoRow("Book", humanizeToken(snapshot.bookType)),
          infoRow("Mode", humanizeToken(snapshot.mode)),
          infoRow("Schema Version", snapshot.schemaVersion || "N/A"),
          infoRow("Created", formatDateTime(snapshot.createdAt))
        )
      )
    ]);
    setChildren(this.raw, [el("h2", textWithIcon("raw", "Snapshot JSON")), resultJsonBlock(snapshot.configJson, "Snapshot payload unavailable.")]);
  }
}

class ManualOverrideForm {
  constructor(app) {
    this.app = app;
    this.error = el("p.form-error");
    this.companyID = el("input.input-control", { type: "text", placeholder: "company id" });
    this.reviewID = el("input.input-control", { type: "text", placeholder: "review id" });
    this.bookType = el("select.select-control",
      el("option", { value: "investing" }, "Investing"),
      el("option", { value: "trading" }, "Trading")
    );
    this.originalAction = el("select.select-control",
      ...["buy", "watch", "hold", "trim", "sell", "reject"].map((value) => el("option", { value }, humanizeToken(value)))
    );
    this.overriddenAction = el("select.select-control",
      ...["buy", "watch", "hold", "trim", "sell", "reject"].map((value) => el("option", { value }, humanizeToken(value)))
    );
    this.overrideBy = el("input.input-control", { type: "text", placeholder: "portfolio-manager" });
    this.reason = el("textarea", { rows: 4, placeholder: "Why are we overriding the default review action?" });
    this.submitButton = createActionButton({
      label: "Submit Override",
      iconName: "save",
      variant: "primary",
      onclick: async (event) => {
        await this.submit(event.currentTarget);
      }
    });
    this.el = el(
      ".panel.detail-panel",
      el(".section-heading", el("h2", textWithIcon("override", "Manual Override")), el("p", "Persist a human override without mutating historical review snapshots.")),
      el(
        ".detail-grid",
        el(".field-block", el("label.label", "Company ID"), this.companyID),
        el(".field-block", el("label.label", "Review ID"), this.reviewID),
        el(".field-block", el("label.label", "Book Type"), this.bookType),
        el(".field-block", el("label.label", "Original Action"), this.originalAction),
        el(".field-block", el("label.label", "Overridden Action"), this.overriddenAction),
        el(".field-block", el("label.label", "Override By"), this.overrideBy)
      ),
      el(".field-block", el("label.label", "Override Reason"), this.reason),
      this.error,
      el(".prompt-actions", this.submitButton)
    );
    this.setError("");
  }

  setBusy(isBusy) {
    [this.companyID, this.reviewID, this.bookType, this.originalAction, this.overriddenAction, this.overrideBy, this.reason, this.submitButton].forEach((field) => {
      field.disabled = isBusy;
    });
  }

  setError(message) {
    this.error.textContent = message || "";
    this.error.style.display = message ? "" : "none";
  }

  applyDraft(draft = null) {
    if (!draft) {
      return;
    }
    this.companyID.value = draft.companyId || "";
    this.reviewID.value = draft.reviewId || "";
    this.bookType.value = draft.bookType || "investing";
    this.originalAction.value = draft.originalAction || "watch";
    this.overriddenAction.value = draft.overriddenAction || draft.originalAction || "hold";
    this.reason.value = draft.overrideReason || "";
    this.overrideBy.value = draft.overrideBy || this.overrideBy.value;
  }

  async submit(button) {
    try {
      this.setError("");
      await this.app.createPlatformOverride({
        companyId: this.companyID.value.trim(),
        reviewId: this.reviewID.value.trim(),
        bookType: this.bookType.value,
        originalAction: this.originalAction.value,
        overriddenAction: this.overriddenAction.value,
        overrideReason: this.reason.value.trim(),
        overrideBy: this.overrideBy.value.trim()
      }, button);
    } catch (error) {
      this.setError(error.message);
    }
  }
}

class PlatformOverridesPage {
  constructor(app) {
    this.app = app;
    this.count = el("p");
    this.form = new ManualOverrideForm(app);
    this.cardsHost = el(".job-list");
    this.empty = el(".empty-state", el("p", "No manual overrides have been recorded yet."));
    this.el = el(
      ".detail-panel",
      this.form.el,
      el(
        ".panel.detail-panel",
        el(".section-heading", el("h2", textWithIcon("override", "Override History")), el(".top-actions", this.count)),
        this.empty,
        this.cardsHost
      )
    );
  }

  setBusy(isBusy) {
    this.form.setBusy(isBusy);
  }

  update(overrides, draft = null) {
    this.form.applyDraft(draft);
    this.count.textContent = `${overrides.length} loaded`;
    this.empty.style.display = overrides.length ? "none" : "";
    setChildren(this.cardsHost, overrides.map((override) => buildOverrideCard(override)));
  }
}

class WelcomePage {
  constructor(app) {
    this.app = app;
    this.copy = el("p.hero-text");
    this.actions = el(".top-actions");
    this.el = el(
      ".panel.welcome-panel",
      el("p.eyebrow", textWithIcon("welcome", "Welcome")),
      el("h1.detail-title", "One workspace for platform operations and async AI workflows."),
      this.copy,
      this.actions
    );
  }

  update(capabilities) {
    const actions = [];

    if (capabilities.platform) {
      actions.push(createNavigateButton(this.app, "/platform", "Open Platform", "platform", { variant: "primary" }));
    }
    if (capabilities.batch) {
      actions.push(createNavigateButton(this.app, "/submissions", "Open Batch App", "submissions"));
    }

    this.copy.textContent =
      capabilities.platform && capabilities.batch
        ? "Use the navigation to move between the investing platform foundation and the existing batch automation workspace."
        : capabilities.platform
          ? "Use the platform routes to inspect companies, review snapshots, theses, workflow runs, allocations, positions, and manual overrides."
          : "Use the batch routes to create prompt batches, procedures, and multi-step executions.";
    setChildren(this.actions, actions);
  }
}

class ProcedureStepField {
  constructor(entry = {}, onRemove) {
    this.onRemove = onRemove;
    this.index = 0;
    this.canRemove = true;
    this.isBusy = false;
    this.label = el("label.label");
    this.copyButton = createCopyButton(() => this.textarea.value);
    this.textarea = el("textarea", {
      rows: 4,
      placeholder: "Describe what this step should do."
    });
    this.textarea.value = entry.prompt || "";
    this.modelSettings = new ModelSettingsControl({
      model: entry.model || DEFAULT_MODEL,
      reasoningEffort: entry.reasoningEffort || ""
    });
    this.removeButton = el(
      "button.button.button-secondary.button-small",
      {
        type: "button",
        onclick: () => this.onRemove(this)
      },
      "Remove"
    );
    setButtonLabel(this.removeButton, "Remove", false, "remove");
    this.el = el(
      ".prompt-card",
      el(".prompt-card-header", this.label, el(".inline-actions", this.copyButton, this.removeButton)),
      this.textarea,
      this.modelSettings.el
    );
    this.update({ index: 0, canRemove: true });
  }

  update({ index, canRemove }) {
    this.index = index;
    this.canRemove = canRemove;
    this.label.textContent = `Step ${index + 1}`;
    this.textarea.id = `procedure-step-${index}`;
    this.label.htmlFor = this.textarea.id;
    this.removeButton.disabled = this.isBusy || !canRemove;
  }

  hasPrompt() {
    return Boolean(this.textarea.value.trim());
  }

  toPayload() {
    return {
      prompt: this.textarea.value,
      model: this.modelSettings.getModel(),
      reasoningEffort: this.modelSettings.getReasoningEffort()
    };
  }

  setBusy(isBusy) {
    this.isBusy = isBusy;
    this.removeButton.disabled = isBusy || !this.canRemove;
    this.copyButton.disabled = isBusy;
    this.textarea.disabled = isBusy;
    this.modelSettings.setBusy(isBusy);
  }
}

class ProcedureDialog {
  constructor(app) {
    this.app = app;
    this.editingProcedureId = null;
    this.fields = [];
    this.title = el("h2.dialog-title", textWithIcon("procedures", "New Procedure"));
    this.error = el("p.form-error");
    this.nameInput = el("input.input-control", {
      type: "text",
      placeholder: "Procedure name"
    });
    this.nameCopyButton = createCopyButton(() => this.nameInput.value);
    this.stepsHost = el(".prompt-stack");
    this.addStepButton = el(
      "button.button.button-secondary",
      {
        type: "button",
        onclick: () => this.addStep()
      },
      "Add Step"
    );
    this.cancelButton = el(
      "button.button.button-secondary",
      {
        type: "button",
        onclick: () => this.close()
      },
      "Cancel"
    );
    this.submitButton = el("button.button.button-primary", { type: "submit" }, "Save Procedure");
    this.closeButton = el(
      "button.button.button-secondary.button-small.dialog-close",
      {
        type: "button",
        onclick: () => this.close()
      },
      "Close"
    );
    setButtonLabel(this.addStepButton, "Add Step", false, "add");
    setButtonLabel(this.cancelButton, "Cancel", false, "close");
    setButtonLabel(this.submitButton, "Save Procedure", false, "save");
    setButtonLabel(this.closeButton, "Close", false, "close");
    this.form = el(
      "form.query-form",
      {
        onsubmit: async (event) => {
          event.preventDefault();
          await this.submit();
        }
      },
      el(
        ".field-block",
        el(".field-label-row", el("label.label", textWithIcon("procedure", "Name")), this.nameCopyButton),
        this.nameInput
      ),
      el(".field-block", el(".dialog-subheading", textWithIcon("steps", "Steps")), this.stepsHost),
      el("p.muted.dialog-help", "Empty steps are ignored when you save."),
      this.error,
      el(".prompt-actions", this.addStepButton, el(".dialog-actions", this.cancelButton, this.submitButton))
    );
    this.dialog = el(
      "dialog.app-dialog",
      el(".dialog-shell", el(".dialog-header", this.title, this.closeButton), this.form)
    );
    this.dialog.addEventListener("cancel", (event) => {
      event.preventDefault();
      this.close();
    });
    this.reset();
  }

  addStep(step = {}) {
    const field = new ProcedureStepField(step, (target) => this.removeStep(target));
    this.fields.push(field);
    this.syncFields();
  }

  removeStep(field) {
    if (this.fields.length === 1) {
      return;
    }

    this.fields = this.fields.filter((item) => item !== field);
    this.syncFields();
  }

  syncFields() {
    this.fields.forEach((field, index) => {
      field.update({
        index,
        canRemove: this.fields.length > 1
      });
    });
    setChildren(this.stepsHost, this.fields);
  }

  open(procedure = null) {
    this.editingProcedureId = procedure?.id || null;
    setChildren(this.title, [textWithIcon(procedure ? "edit" : "procedures", procedure ? "Edit Procedure" : "New Procedure")]);
    setButtonLabel(this.submitButton, procedure ? "Update Procedure" : "Save Procedure", false, "save");
    this.nameInput.value = procedure?.name || "";
    this.fields = [];
    const steps = Array.isArray(procedure?.steps) && procedure.steps.length ? procedure.steps : [{}];

    steps.forEach((step) => this.addStep(step));
    this.setError("");
    this.dialog.showModal();
  }

  reset() {
    this.editingProcedureId = null;
    this.nameInput.value = "";
    this.fields = [];
    this.addStep();
    this.setError("");
  }

  close() {
    if (this.dialog.open) {
      this.dialog.close();
    }
    this.reset();
  }

  setError(message) {
    this.error.textContent = message || "";
    this.error.style.display = message ? "" : "none";
  }

  setBusy(isBusy) {
    this.nameInput.disabled = isBusy;
    this.nameCopyButton.disabled = isBusy;
    this.addStepButton.disabled = isBusy;
    this.cancelButton.disabled = isBusy;
    this.submitButton.disabled = isBusy;
    this.closeButton.disabled = isBusy;
    this.fields.forEach((field) => field.setBusy(isBusy));
  }

  async submit() {
    const isEditing = Boolean(this.editingProcedureId);

    try {
      this.setError("");
      this.app.setBusy(true);
      setButtonLabel(this.submitButton, isEditing ? "Update Procedure" : "Save Procedure", true, "save");
      const steps = this.fields.map((field) => field.toPayload());
      const payload = isEditing
        ? await apiFetch(`/api/procedures/${this.editingProcedureId}`, {
            method: "PUT",
            body: JSON.stringify({
              name: this.nameInput.value,
              steps
            })
          })
        : await apiFetch("/api/procedures", {
            method: "POST",
            body: JSON.stringify({
              name: this.nameInput.value,
              steps
            })
          });

      await this.app.loadProcedures();
      this.app.setMessage(`${payload.procedure.name} saved.`);
      this.close();
    } catch (error) {
      this.setError(error.message);
    } finally {
      setButtonLabel(this.submitButton, isEditing ? "Update Procedure" : "Save Procedure", false, "save");
      this.app.setBusy(false);
    }
  }
}

class ProcedureCard {
  constructor(app) {
    this.app = app;
    this.name = el("h3.job-title");
    this.meta = el(".job-meta");
    this.preview = el("p.muted.card-preview");
    this.editButton = el(
      "button.button.button-secondary.button-small",
      {
        type: "button",
        onclick: () => this.app.openProcedureDialog(this.procedure)
      },
      "Edit"
    );
    setButtonLabel(this.editButton, "Edit", false, "edit");
    this.el = el(
      "article.job-card",
      el(".job-header", this.name, this.editButton),
      this.meta,
      this.preview
    );
  }

  update(procedure) {
    this.procedure = procedure;
    setChildren(this.name, [textWithIcon("procedure", procedure.name)]);
    this.meta.textContent = `${procedure.stepCount} step${procedure.stepCount === 1 ? "" : "s"} • Updated ${procedure.updatedAtLabel}`;
    const previewText = procedure.steps.map((step) => step.prompt).join(" | ");
    this.preview.textContent = previewText || "No steps yet.";
  }
}

class ProceduresPage {
  constructor(app) {
    this.app = app;
    this.count = el("p");
    this.cardsHost = el(".job-list");
    this.cards = [];
    this.addButton = el(
      "button.button.button-primary",
      {
        type: "button",
        onclick: () => this.app.openProcedureDialog()
      },
      "Add Procedure"
    );
    setButtonLabel(this.addButton, "Add Procedure", false, "add");
    this.empty = el(".empty-state", el("p", "No procedures yet. Create one to define multi-step work."));
    this.el = el(
      ".panel",
      el(".section-heading", el("h2", textWithIcon("procedures", "Procedures")), el(".top-actions", this.count, this.addButton)),
      this.empty,
      this.cardsHost
    );
  }

  setBusy(isBusy) {
    this.addButton.disabled = isBusy;
  }

  update(procedures) {
    this.count.textContent = `${procedures.length} total`;
    this.empty.style.display = procedures.length ? "none" : "";
    this.cards = procedures.map((procedure, index) => {
      const card = this.cards[index] || new ProcedureCard(this.app);
      card.update(procedure);
      return card;
    });
    setChildren(this.cardsHost, this.cards);
  }
}

class ExecutionDialog {
  constructor(app) {
    this.app = app;
    this.error = el("p.form-error");
    this.title = el("h2.dialog-title", textWithIcon("execution", "Create Execution"));
    this.procedureSelect = el("select.select-control");
    this.promptInput = el("textarea", {
      rows: 6,
      placeholder: "Initial prompt or input for the procedure."
    });
    this.promptCopyButton = createCopyButton(() => this.promptInput.value);
    this.cancelButton = el(
      "button.button.button-secondary",
      {
        type: "button",
        onclick: () => this.close()
      },
      "Cancel"
    );
    this.submitButton = el("button.button.button-primary", { type: "submit" }, "Create & Start");
    this.closeButton = el(
      "button.button.button-secondary.button-small.dialog-close",
      {
        type: "button",
        onclick: () => this.close()
      },
      "Close"
    );
    setButtonLabel(this.cancelButton, "Cancel", false, "close");
    setButtonLabel(this.submitButton, "Create & Start", false, "start");
    setButtonLabel(this.closeButton, "Close", false, "close");
    this.form = el(
      "form.query-form",
      {
        onsubmit: async (event) => {
          event.preventDefault();
          await this.submit();
        }
      },
      el(".field-block", el("label.label", textWithIcon("procedure", "Procedure")), this.procedureSelect),
      el(
        ".field-block",
        el(".field-label-row", el("label.label", textWithIcon("prompt", "Prompt")), this.promptCopyButton),
        this.promptInput
      ),
      this.error,
      el(".prompt-actions", el(".dialog-actions", this.cancelButton, this.submitButton))
    );
    this.dialog = el(
      "dialog.app-dialog",
      el(".dialog-shell", el(".dialog-header", this.title, this.closeButton), this.form)
    );
    this.dialog.addEventListener("cancel", (event) => {
      event.preventDefault();
      this.close();
    });
  }

  open(procedures) {
    setChildren(
      this.procedureSelect,
      (procedures || []).map((procedure) => el("option", { value: procedure.id }, procedure.name))
    );
    this.setError("");
    this.promptInput.value = "";
    this.dialog.showModal();
  }

  close() {
    if (this.dialog.open) {
      this.dialog.close();
    }
    this.setError("");
    this.promptInput.value = "";
  }

  setError(message) {
    this.error.textContent = message || "";
    this.error.style.display = message ? "" : "none";
  }

  setBusy(isBusy) {
    this.procedureSelect.disabled = isBusy;
    this.promptInput.disabled = isBusy;
    this.promptCopyButton.disabled = isBusy;
    this.cancelButton.disabled = isBusy;
    this.submitButton.disabled = isBusy;
    this.closeButton.disabled = isBusy;
  }

  async submit() {
    try {
      this.setError("");
      this.app.setBusy(true);
      setButtonLabel(this.submitButton, "Create & Start", true, "start");
      const payload = await apiFetch("/api/procedure-executions", {
        method: "POST",
        body: JSON.stringify({
          procedureId: this.procedureSelect.value,
          prompt: this.promptInput.value
        })
      });

      await this.app.loadExecutions();
      this.app.setMessage("Execution created and started.");
      this.close();
      this.app.navigate(pathForExecution(payload.execution.id));
    } catch (error) {
      this.setError(error.message);
    } finally {
      setButtonLabel(this.submitButton, "Create & Start", false, "start");
      this.app.setBusy(false);
    }
  }
}

class ExecutionCard {
  constructor(app) {
    this.app = app;
    this.status = el("span.status-pill");
    this.titleLink = el("a.job-link", {
      href: "/procedure-executions",
      onclick: (event) => {
        event.preventDefault();
        this.app.navigate(pathForExecution(this.execution.id));
      }
    });
    this.meta = el(".job-meta");
    this.preview = el("p.muted.card-preview");
    this.el = el("article.job-card", el(".job-header", this.status, this.titleLink), this.meta, this.preview);
  }

  update(execution) {
    this.execution = execution;
    this.status.className = `status-pill ${statusClassName(execution.status)}`;
    this.status.textContent = execution.status;
    this.titleLink.href = pathForExecution(execution.id);
    this.titleLink.textContent = execution.procedureName;
    this.meta.textContent = `Created ${execution.createdAtLabel}`;
    this.preview.textContent = execution.initialPromptLabel;
  }
}

class ProcedureExecutionsPage {
  constructor(app) {
    this.app = app;
    this.count = el("p");
    this.cardsHost = el(".job-list");
    this.cards = [];
    this.createButton = el(
      "button.button.button-primary",
      {
        type: "button",
        onclick: () => this.app.openExecutionDialog()
      },
      "Create Execution"
    );
    setButtonLabel(this.createButton, "Create Execution", false, "add");
    this.empty = el(".empty-state", el("p", "No executions yet. Create one from a saved procedure."));
    this.el = el(
      ".panel",
      el(".section-heading", el("h2", textWithIcon("executions", "Procedure Executions")), el(".top-actions", this.count, this.createButton)),
      this.empty,
      this.cardsHost
    );
  }

  setBusy(isBusy) {
    this.createButton.disabled = isBusy || !this.app.state.procedures.length;
  }

  update(executions) {
    this.count.textContent = `${executions.length} total`;
    this.empty.style.display = executions.length ? "none" : "";
    this.cards = executions.map((execution, index) => {
      const card = this.cards[index] || new ExecutionCard(this.app);
      card.update(execution);
      return card;
    });
    setChildren(this.cardsHost, this.cards);
  }
}

class ExecutionStepCard {
  constructor(app) {
    this.app = app;
    this.disclosure = el("details.execution-step-card");
    this.summary = el("summary.execution-step-summary");
    this.summaryCopy = el(".execution-step-summary-copy");
    this.refreshButton = el(
      "button.button.button-secondary.button-small",
      {
        type: "button",
        onclick: async (event) => {
          event.preventDefault();
          event.stopPropagation();
          await this.app.refreshExecution(this.executionId, { button: event.currentTarget });
        }
      },
      "Refresh"
    );
    setButtonLabel(this.refreshButton, "Refresh", false, "refresh");
    this.content = el(".execution-step-content");
    this.disclosure.open = false;
    setChildren(this.disclosure, [this.summary, this.content]);
    this.el = this.disclosure;
  }

  update(executionId, step) {
    this.executionId = executionId;
    const thinkingLabel = step.reasoningEffort ? formatReasoningLabel(step.reasoningEffort) : "N/A";
    this.refreshButton.style.display = step.canRefresh ? "" : "none";
    setChildren(this.summaryCopy, [
      el("span.execution-step-number", `Step ${step.stepNumber}`),
      el("span.execution-step-prompt", step.prompt || "N/A"),
      el("span.execution-step-meta", `Model: ${step.model} • Thinking: ${thinkingLabel}`)
    ]);
    setChildren(this.summary, [this.summaryCopy, this.refreshButton]);

    const inputDisclosure = el("details.raw-disclosure", { open: false }, el("summary.raw-summary", textWithIcon("input", "Input")));
    const inputChildren = step.stepInput
      ? [buildCopyableSurface(el(".result-box", el("pre", step.stepInput)), () => step.stepInput)]
      : [el("p.muted", "Input will appear once this step starts.")];
    setChildren(inputDisclosure, [el("summary.raw-summary", textWithIcon("input", "Input")), ...inputChildren]);

    const outputDisclosure = el("details.raw-disclosure", { open: false }, el("summary.raw-summary", textWithIcon("output", "Output")));
    const outputChildren = [];

    if (step.resultText) {
      const markdownOutput = el(".markdown-output");
      markdownOutput.innerHTML = marked.parse(step.resultText);
      outputChildren.push(buildCopyableSurface(el(".result-box.markdown-result", markdownOutput), () => step.resultText));
    } else {
      outputChildren.push(el("p.muted", "No output yet."));
    }

    setChildren(outputDisclosure, [el("summary.raw-summary", textWithIcon("output", "Output")), ...outputChildren]);

    const rawResponseDisclosure = step.resultResponseBody
      ? el(
          "details.raw-disclosure",
          { open: false },
          el("summary.raw-summary", textWithIcon("raw", "Raw Response")),
          buildCopyableSurface(el(".result-box", el("pre", formatJson(step.resultResponseBody))), () =>
            formatJson(step.resultResponseBody)
          )
        )
      : null;
    const info = el(
      ".detail-list",
      this.infoRow("Status", el("span.status-pill", { className: `status-pill ${statusClassName(step.status)}` }, step.status)),
      this.infoRow("Model", step.model),
      this.infoRow("Thinking", thinkingLabel),
      this.infoRow("Started", step.startedAtLabel),
      this.infoRow("Completed", step.completedAtLabel || "Not yet"),
      this.infoRow("Execution Time", step.durationLabel),
      this.infoRow("Batch ID", el("code", step.batchId || "pending"))
    );

    const blocks = [info, inputDisclosure, outputDisclosure];

    if (rawResponseDisclosure) {
      blocks.push(rawResponseDisclosure);
    }

    if (step.latestError) {
      blocks.push(
        el(
          ".detail-block",
          el("h3", textWithIcon("error", "Error")),
          buildCopyableSurface(el(".result-box.result-box-error", el("pre", formatJson(step.latestError))), () =>
            formatJson(step.latestError)
          )
        )
      );
    }

    setChildren(this.content, blocks);
  }

  infoRow(label, value) {
    return el("div", el("dt", label), el("dd", value));
  }
}

class ProcedureExecutionDetail {
  constructor(app) {
    this.app = app;
    this.topline = el(".detail-topline");
    this.title = el("h1.detail-title");
    this.info = el(".detail-list");
    this.initialInput = el(".detail-block");
    this.meta = el(".detail-block");
    this.stepsHost = el(".execution-steps-list");
    this.stepsBlock = el(".detail-block", el("h2", textWithIcon("steps", "Steps")), this.stepsHost);
    this.stepCards = [];
    this.el = el(".panel.detail-panel", this.topline, this.title, this.initialInput, this.meta, this.stepsBlock);
  }

  infoRow(label, value) {
    return el("div", el("dt", label), el("dd", value));
  }

  update(execution) {
    const backButton = el(
      "button.button.button-secondary.button-small",
      {
        type: "button",
        onclick: () => this.app.navigate("/procedure-executions")
      },
      "Back"
    );
    setButtonLabel(backButton, "Back", false, "back");
    const status = el("span.status-pill", { className: `status-pill ${statusClassName(execution.status)}` }, execution.status);
    const actions = [backButton, status];

    if (execution.status === "draft") {
      const startButton = el(
        "button.button.button-primary.button-small",
        {
          type: "button",
          onclick: async (event) => {
            await this.app.startExecution(execution.id, { button: event.currentTarget });
          }
        },
        "Start"
      );
      setButtonLabel(startButton, "Start", false, "start");
      actions.push(startButton);
    }

    setChildren(this.topline, actions);
    setChildren(this.title, [textWithIcon("execution", execution.procedureName)]);

    const inputDisclosure = el("details.raw-disclosure");
    inputDisclosure.open = false;
    setChildren(inputDisclosure, [
      el("summary.raw-summary", textWithIcon("input", "Initial Input")),
      buildCopyableSurface(el(".result-box", el("pre", execution.initialPromptLabel)), () => execution.initialPrompt || "")
    ]);
    setChildren(this.initialInput, [el("h2", textWithIcon("input", "Initial Input")), inputDisclosure]);

    setChildren(this.meta, [
      el("h2", textWithIcon("details", "Execution Details")),
      el(
        ".detail-list",
        this.infoRow("Procedure", execution.procedureName),
        this.infoRow("Created", execution.createdAtLabel),
        this.infoRow("Started", execution.startedAtLabel || "Not started"),
        this.infoRow("Completed", execution.completedAtLabel || "Not yet"),
        this.infoRow("Current Step", execution.currentStepNumber || "N/A")
      )
    ]);

    this.stepCards = execution.steps.map((step, index) => {
      const card = this.stepCards[index] || new ExecutionStepCard(this.app);
      card.update(execution.id, step);
      return card;
    });
    setChildren(this.stepsHost, this.stepCards);
  }
}

class App {
  constructor() {
    this.state = {
      capabilitiesLoaded: false,
      capabilities: {
        platform: false,
        batch: false
      },
      submissions: [],
      procedures: [],
      executions: [],
      currentSubmission: null,
      currentExecution: null,
      platformCompanies: [],
      platformCompanySearch: "",
      currentCompany: null,
      currentCompanyReviews: [],
      currentCompanyThesis: null,
      currentCompanyHistorySummary: null,
      platformReviews: [],
      platformReviewBookType: "",
      currentReview: null,
      currentReviewDiff: null,
      currentReviewEvidence: [],
      platformWorkflowRuns: [],
      currentWorkflowRun: null,
      currentWorkflowSummary: null,
      currentWorkflowStatus: null,
      currentWorkflowSteps: [],
      currentWorkflowBatchJobs: [],
      platformAIBatchJobs: [],
      currentAIBatchJob: null,
      currentAIBatchItems: [],
      platformCapitalAllocations: [],
      currentCapitalAllocation: null,
      platformPositions: [],
      platformPositionsBookType: "",
      currentConfig: null,
      platformConfigSnapshots: [],
      currentConfigSnapshot: null,
      platformOverrides: [],
      platformOverrideDraft: null,
      sidebarOpen: true,
      busy: false,
      message: ""
    };
    this.executionRefreshInFlight = false;

    this.sidebar = new Sidebar(this);
    this.overlay = el(".sidebar-overlay", {
      onclick: () => this.closeSidebar()
    });
    this.navToggle = el(
      "button.button.button-secondary",
      {
        type: "button",
        onclick: () => this.toggleSidebar()
      },
      "Menu"
    );
    setButtonLabel(this.navToggle, "Menu", false, "menu");
    this.refreshAllButton = el(
      "button.button.button-secondary",
      {
        type: "button",
        onclick: async () => {
          await this.refreshAll();
        }
      },
      "Refresh All"
    );
    setButtonLabel(this.refreshAllButton, "Refresh All", false, "refresh");
    this.message = el("p.form-error");
    this.pageHost = el(".page-content");
    this.manualForm = new ManualSubmissionForm(this);
    this.templatedForm = new TemplatedSubmissionForm(this);
    this.submissionsList = new SubmissionsList(this);
    this.submissionDetail = new SubmissionDetail(this);
    this.procedureDialog = new ProcedureDialog(this);
    this.executionDialog = new ExecutionDialog(this);
    this.proceduresPage = new ProceduresPage(this);
    this.procedureExecutionsPage = new ProcedureExecutionsPage(this);
    this.procedureExecutionDetail = new ProcedureExecutionDetail(this);
    this.platformHomePage = new PlatformHomePage(this);
    this.platformCompaniesPage = new PlatformCompaniesPage(this);
    this.platformCompanyDetailPage = new PlatformCompanyDetailPage(this);
    this.platformReviewsPage = new PlatformReviewsPage(this);
    this.platformReviewDetailPage = new PlatformReviewDetailPage(this);
    this.platformWorkflowRunsPage = new PlatformWorkflowRunsPage(this);
    this.platformWorkflowRunDetailPage = new PlatformWorkflowRunDetailPage(this);
    this.platformAIBatchJobsPage = new PlatformAIBatchJobsPage(this);
    this.platformAIBatchJobDetailPage = new PlatformAIBatchJobDetailPage(this);
    this.platformCapitalAllocationsPage = new PlatformCapitalAllocationsPage(this);
    this.platformCapitalAllocationDetailPage = new PlatformCapitalAllocationDetailPage(this);
    this.platformPositionsPage = new PlatformPositionsPage(this);
    this.platformConfigPage = new PlatformConfigPage(this);
    this.platformConfigSnapshotDetailPage = new PlatformConfigSnapshotDetailPage(this);
    this.platformOverridesPage = new PlatformOverridesPage(this);
    this.unavailablePage = new UnavailablePage(this);
    this.welcomePage = new WelcomePage(this);
    this.shell = el(
      ".spa-shell",
      this.sidebar.el,
      this.overlay,
      el(".main-shell", el(".top-toolbar", this.navToggle, this.refreshAllButton), this.message, this.pageHost),
      this.procedureDialog.dialog,
      this.executionDialog.dialog
    );
    this.el = this.shell;

    window.addEventListener("popstate", () => this.renderRoute());
    this.startThemeSchedule();
    this.startExecutionRefreshSchedule();
    this.setMessage("");
  }

  setBusy(isBusy) {
    this.state.busy = isBusy;
    this.navToggle.disabled = isBusy;
    this.refreshAllButton.disabled = isBusy;
    this.manualForm.setBusy(isBusy);
    this.templatedForm.setBusy(isBusy);
    this.procedureDialog.setBusy(isBusy);
    this.executionDialog.setBusy(isBusy);
    this.proceduresPage.setBusy(isBusy);
    this.procedureExecutionsPage.setBusy(isBusy);
    this.platformCompaniesPage.setBusy(isBusy);
    this.platformWorkflowRunsPage.setBusy(isBusy);
    this.platformOverridesPage.setBusy(isBusy);
  }

  setMessage(message) {
    this.state.message = message || "";
    this.message.textContent = this.state.message;
    this.message.style.display = this.state.message ? "" : "none";
  }

  toggleSidebar() {
    this.state.sidebarOpen = !this.state.sidebarOpen;
    this.renderChrome();
  }

  closeSidebar() {
    this.state.sidebarOpen = false;
    this.renderChrome();
  }

  getNavigationItems() {
    return [
      ...BASE_NAV_ITEMS,
      ...(this.state.capabilities.platform ? PLATFORM_NAV_ITEMS : []),
      ...(this.state.capabilities.batch ? BATCH_NAV_ITEMS : [])
    ];
  }

  async ensureCapabilities(force = false) {
    if (this.state.capabilitiesLoaded && !force) {
      return this.state.capabilities;
    }

    const [platform, batch] = await Promise.all([
      detectCapability("/api/v1/config/current"),
      detectCapability("/api/submissions")
    ]);

    this.state.capabilitiesLoaded = true;
    this.state.capabilities = { platform, batch };
    this.welcomePage.update(this.state.capabilities);
    return this.state.capabilities;
  }

  renderChrome() {
    const route = getRoute();
    const currentPath = primaryPathForRoute(route);
    const isSubmissionRoute = route.page === "submissions" || route.page === "submission-detail";
    this.sidebar.update(currentPath, this.state.sidebarOpen, this.getNavigationItems());
    this.overlay.className = `sidebar-overlay${this.state.sidebarOpen ? " sidebar-overlay-visible" : ""}`;
    this.refreshAllButton.style.display = isSubmissionRoute ? "" : "none";
  }

  navigate(path) {
    window.history.pushState({}, "", path);
    this.closeSidebar();
    this.renderRoute();
  }

  async loadSubmissions() {
    const payload = await apiFetch("/api/submissions");
    this.state.submissions = payload.jobs;
    this.submissionsList.update(this.state.submissions);
    const refreshableCount = this.state.submissions.filter((job) => job.canRefresh).length;
    setButtonLabel(this.refreshAllButton, refreshableCount ? `Refresh All (${refreshableCount})` : "Refresh All", false, "refresh");
    return this.state.submissions;
  }

  async loadProcedures() {
    const payload = await apiFetch("/api/procedures");
    this.state.procedures = payload.procedures;
    this.proceduresPage.update(this.state.procedures);
    this.procedureExecutionsPage.setBusy(this.state.busy);
    return this.state.procedures;
  }

  async loadExecutions() {
    const payload = await apiFetch("/api/procedure-executions");
    this.state.executions = payload.executions;
    this.procedureExecutionsPage.update(this.state.executions);
    return this.state.executions;
  }

  async loadSubmission(id) {
    const payload = await apiFetch(`/api/submissions/${id}`);
    this.state.currentSubmission = payload.job;
    this.submissionDetail.update(payload.job);
    return payload.job;
  }

  async loadExecution(id) {
    const payload = await apiFetch(`/api/procedure-executions/${id}`);
    this.state.currentExecution = payload.execution;
    this.procedureExecutionDetail.update(payload.execution);
    return payload.execution;
  }

  async loadPlatformCompanies(search = this.state.platformCompanySearch) {
    const query = new URLSearchParams();
    query.set("limit", "50");
    if (search) {
      query.set("search", search);
    }
    const payload = await apiFetch(`/api/v1/companies?${query.toString()}`);
    this.state.platformCompanies = safeArray(payload.companies);
    this.state.platformCompanySearch = search;
    this.platformCompaniesPage.setError("");
    this.platformCompaniesPage.update(this.state.platformCompanies, search);
    return this.state.platformCompanies;
  }

  async loadPlatformCompany(id) {
    const [companyPayload, reviewsPayload, thesisPayload, historyPayload] = await Promise.all([
      apiFetch(`/api/v1/companies/${id}`),
      apiFetch(`/api/v1/companies/${id}/reviews?limit=12`),
      apiFetchOptional(`/api/v1/companies/${id}/thesis`),
      apiFetch(`/api/v1/companies/${id}/history-summary`)
    ]);

    this.state.currentCompany = companyPayload.company;
    this.state.currentCompanyReviews = safeArray(reviewsPayload?.reviews);
    this.state.currentCompanyThesis = thesisPayload?.thesis || null;
    this.state.currentCompanyHistorySummary = historyPayload?.summary || null;
    this.platformCompanyDetailPage.update(
      this.state.currentCompany,
      this.state.currentCompanyReviews,
      this.state.currentCompanyThesis,
      this.state.currentCompanyHistorySummary
    );
    return this.state.currentCompany;
  }

  async loadPlatformReviews(bookType = this.state.platformReviewBookType) {
    const query = new URLSearchParams();
    query.set("limit", "50");
    if (bookType) {
      query.set("book_type", bookType);
    }
    const payload = await apiFetch(`/api/v1/reviews?${query.toString()}`);
    this.state.platformReviews = safeArray(payload.reviews);
    this.state.platformReviewBookType = bookType;
    this.platformReviewsPage.update(this.state.platformReviews, bookType);
    return this.state.platformReviews;
  }

  async loadPlatformReview(id) {
    const [reviewPayload, diffPayload, evidencePayload] = await Promise.all([
      apiFetch(`/api/v1/reviews/${id}`),
      apiFetchOptional(`/api/v1/reviews/${id}/diff`),
      apiFetchOptional(`/api/v1/reviews/${id}/evidence`)
    ]);

    this.state.currentReview = reviewPayload.review;
    this.state.currentReviewDiff = diffPayload?.diff || null;
    this.state.currentReviewEvidence = safeArray(evidencePayload?.evidence);
    this.platformReviewDetailPage.update(
      this.state.currentReview,
      this.state.currentReviewDiff,
      this.state.currentReviewEvidence
    );
    return this.state.currentReview;
  }

  async loadPlatformWorkflowRuns() {
    const payload = await apiFetch("/api/v1/workflow-runs?limit=50");
    this.state.platformWorkflowRuns = safeArray(payload.workflowRuns);
    this.platformWorkflowRunsPage.update(this.state.platformWorkflowRuns);
    return this.state.platformWorkflowRuns;
  }

  async loadPlatformWorkflowRun(id) {
    const [runPayload, summaryPayload, statusPayload, stepsPayload, batchJobsPayload] = await Promise.all([
      apiFetch(`/api/v1/workflow-runs/${id}`),
      apiFetchOptional(`/api/v1/workflow-runs/${id}/summary`),
      apiFetchOptional(`/api/v1/workflow-runs/${id}/status`),
      apiFetchOptional(`/api/v1/workflow-runs/${id}/steps`),
      apiFetchOptional(`/api/v1/ai-batch-jobs?workflow_run_id=${encodeURIComponent(id)}&limit=50`)
    ]);

    this.state.currentWorkflowRun = runPayload.workflowRun;
    this.state.currentWorkflowSummary = summaryPayload?.summary || null;
    this.state.currentWorkflowStatus = statusPayload?.status || null;
    this.state.currentWorkflowSteps = safeArray(stepsPayload?.workflowSteps || statusPayload?.status?.steps);
    this.state.currentWorkflowBatchJobs = safeArray(batchJobsPayload?.aiBatchJobs);
    this.platformWorkflowRunDetailPage.update(
      this.state.currentWorkflowRun,
      this.state.currentWorkflowSummary,
      this.state.currentWorkflowStatus,
      this.state.currentWorkflowSteps,
      this.state.currentWorkflowBatchJobs
    );
    return this.state.currentWorkflowRun;
  }

  async loadPlatformAIBatchJobs() {
    const payload = await apiFetch("/api/v1/ai-batch-jobs?limit=50");
    this.state.platformAIBatchJobs = safeArray(payload.aiBatchJobs);
    this.platformAIBatchJobsPage.update(this.state.platformAIBatchJobs);
    return this.state.platformAIBatchJobs;
  }

  async loadPlatformAIBatchJob(id) {
    const [jobPayload, itemsPayload] = await Promise.all([
      apiFetch(`/api/v1/ai-batch-jobs/${id}`),
      apiFetch(`/api/v1/ai-batch-jobs/${id}/items?limit=200`)
    ]);

    this.state.currentAIBatchJob = jobPayload.aiBatchJob;
    this.state.currentAIBatchItems = safeArray(itemsPayload.aiBatchItems);
    this.platformAIBatchJobDetailPage.update(this.state.currentAIBatchJob, this.state.currentAIBatchItems);
    return this.state.currentAIBatchJob;
  }

  async loadPlatformCapitalAllocations() {
    const payload = await apiFetch("/api/v1/capital-allocations?limit=50");
    this.state.platformCapitalAllocations = safeArray(payload.capitalAllocations);
    this.platformCapitalAllocationsPage.update(this.state.platformCapitalAllocations);
    return this.state.platformCapitalAllocations;
  }

  async loadPlatformCapitalAllocation(id) {
    const payload = await apiFetch(`/api/v1/capital-allocations/${id}`);
    this.state.currentCapitalAllocation = payload.capitalAllocation;
    this.platformCapitalAllocationDetailPage.update(this.state.currentCapitalAllocation);
    return this.state.currentCapitalAllocation;
  }

  async loadPlatformPositions(bookType = this.state.platformPositionsBookType) {
    const endpoint = bookType ? `/api/v1/positions/${bookType}` : "/api/v1/positions";
    const payload = await apiFetch(`${endpoint}?limit=50`);
    this.state.platformPositions = safeArray(payload.positions);
    this.state.platformPositionsBookType = bookType;
    this.platformPositionsPage.update(this.state.platformPositions, bookType);
    return this.state.platformPositions;
  }

  async loadPlatformConfig() {
    const [configPayload, snapshotsPayload] = await Promise.all([
      apiFetch("/api/v1/config/current"),
      apiFetch("/api/v1/config/snapshots?limit=50")
    ]);
    this.state.currentConfig = configPayload.config || null;
    this.state.platformConfigSnapshots = safeArray(snapshotsPayload.configSnapshots);
    this.platformConfigPage.update(this.state.currentConfig, this.state.platformConfigSnapshots);
    this.platformHomePage.update(this.state);
    return this.state.currentConfig;
  }

  async loadPlatformConfigSnapshot(id) {
    const payload = await apiFetch(`/api/v1/config/snapshots/${id}`);
    this.state.currentConfigSnapshot = payload.configSnapshot;
    this.platformConfigSnapshotDetailPage.update(this.state.currentConfigSnapshot);
    return this.state.currentConfigSnapshot;
  }

  async loadPlatformOverrides() {
    const payload = await apiFetch("/api/v1/overrides?limit=50");
    this.state.platformOverrides = safeArray(payload.overrides);
    this.platformOverridesPage.update(this.state.platformOverrides, this.state.platformOverrideDraft);
    return this.state.platformOverrides;
  }

  prepareOverrideFromReview(review) {
    this.state.platformOverrideDraft = {
      companyId: review.companyId,
      reviewId: review.id,
      bookType: review.bookType,
      originalAction: review.finalActionAfterReview || "watch",
      overriddenAction: review.ownedBeforeReview ? "hold" : "watch"
    };
    this.navigate("/platform/overrides");
  }

  async createPlatformOverride(payload, button) {
    try {
      this.setBusy(true);
      this.setMessage("");
      if (button) {
        setButtonLabel(button, button.dataset.label || "Submit Override", true, button.dataset.icon || "save");
      }
      const response = await apiFetch("/api/v1/overrides", {
        method: "POST",
        body: JSON.stringify(payload)
      });
      this.state.platformOverrideDraft = null;
      await this.loadPlatformOverrides();
      this.setMessage(`Override ${response.override.id} saved.`);
      return response.override;
    } finally {
      if (button) {
        setButtonLabel(button, button.dataset.label || "Submit Override", false, button.dataset.icon || "save");
      }
      this.setBusy(false);
    }
  }

  async startPlatformInvestingWorkflow(payload, dryRun, button) {
    try {
      this.setBusy(true);
      this.setMessage("");
      if (button) {
        setButtonLabel(button, button.dataset.label || (dryRun ? "Dry Run" : "Start Async Run"), true, button.dataset.icon || "workflow");
      }
      const response = await apiFetch(dryRun ? "/api/v1/workflow-runs/investing/dry-run" : "/api/v1/workflow-runs/investing/start", {
        method: "POST",
        body: JSON.stringify(payload)
      });
      await this.loadPlatformWorkflowRuns();
      this.setMessage(dryRun ? "Investing dry run created." : "Investing async workflow created.");
      this.navigate(pathForPlatformWorkflowRun(response.workflowRun.id));
      return response.workflowRun;
    } finally {
      if (button) {
        setButtonLabel(button, button.dataset.label || (dryRun ? "Dry Run" : "Start Async Run"), false, button.dataset.icon || "workflow");
      }
      this.setBusy(false);
    }
  }

  async resumePlatformWorkflowRun(id, button) {
    try {
      this.setBusy(true);
      this.setMessage("");
      if (button) {
        setButtonLabel(button, button.dataset.label || "Resume", true, button.dataset.icon || "start");
      }
      const response = await apiFetch(`/api/v1/workflow-runs/${id}/resume`, {
        method: "POST"
      });
      await Promise.all([this.loadPlatformWorkflowRuns(), this.loadPlatformWorkflowRun(id)]);
      this.setMessage(`Workflow ${response.workflowRun.id} resumed.`);
      return response.workflowRun;
    } finally {
      if (button) {
        setButtonLabel(button, button.dataset.label || "Resume", false, button.dataset.icon || "start");
      }
      this.setBusy(false);
    }
  }

  async reconcilePlatformWorkflowRun(id, button) {
    try {
      this.setBusy(true);
      this.setMessage("");
      if (button) {
        setButtonLabel(button, button.dataset.label || "Reconcile", true, button.dataset.icon || "refresh");
      }
      const response = await apiFetch(`/api/v1/workflow-runs/${id}/reconcile`, {
        method: "POST"
      });
      await Promise.all([this.loadPlatformWorkflowRuns(), this.loadPlatformWorkflowRun(id)]);
      this.setMessage(`Workflow ${response.workflowRun.id} reconciled.`);
      return response.workflowRun;
    } finally {
      if (button) {
        setButtonLabel(button, button.dataset.label || "Reconcile", false, button.dataset.icon || "refresh");
      }
      this.setBusy(false);
    }
  }

  async retryPlatformAIBatchJob(id, button) {
    try {
      this.setBusy(true);
      this.setMessage("");
      if (button) {
        setButtonLabel(button, button.dataset.label || "Retry Job", true, button.dataset.icon || "refresh");
      }
      const response = await apiFetch(`/api/v1/ai-batch-jobs/${id}/retry`, {
        method: "POST"
      });
      await Promise.all([this.loadPlatformAIBatchJobs(), this.loadPlatformAIBatchJob(id)]);
      this.setMessage(`AI batch job ${response.aiBatchJob.id} requeued.`);
      return response.aiBatchJob;
    } finally {
      if (button) {
        setButtonLabel(button, button.dataset.label || "Retry Job", false, button.dataset.icon || "refresh");
      }
      this.setBusy(false);
    }
  }

  async retryPlatformAIBatchItem(id, jobID, button) {
    try {
      this.setBusy(true);
      this.setMessage("");
      if (button) {
        setButtonLabel(button, button.dataset.label || "Retry Item", true, button.dataset.icon || "refresh");
      }
      const response = await apiFetch(`/api/v1/ai-batch-items/${id}/retry`, {
        method: "POST"
      });
      await Promise.all([this.loadPlatformAIBatchJobs(), this.loadPlatformAIBatchJob(jobID)]);
      this.setMessage(`AI batch item ${response.aiBatchItem.id} requeued.`);
      return response.aiBatchItem;
    } finally {
      if (button) {
        setButtonLabel(button, button.dataset.label || "Retry Item", false, button.dataset.icon || "refresh");
      }
      this.setBusy(false);
    }
  }

  async skipPlatformAIBatchItem(id, jobID, button) {
    try {
      this.setBusy(true);
      this.setMessage("");
      if (button) {
        setButtonLabel(button, button.dataset.label || "Skip Item", true, button.dataset.icon || "close");
      }
      const response = await apiFetch(`/api/v1/ai-batch-items/${id}/skip`, {
        method: "POST"
      });
      await Promise.all([this.loadPlatformAIBatchJobs(), this.loadPlatformAIBatchJob(jobID)]);
      this.setMessage(`AI batch item ${response.aiBatchItem.id} skipped.`);
      return response.aiBatchItem;
    } finally {
      if (button) {
        setButtonLabel(button, button.dataset.label || "Skip Item", false, button.dataset.icon || "close");
      }
      this.setBusy(false);
    }
  }

  async refreshSubmission(id, options = {}) {
    try {
      this.setBusy(true);
      this.setMessage("");
      if (options.button) {
        setButtonLabel(options.button, options.button.dataset.label || "Refresh", true, options.button.dataset.icon || "refresh");
      }
      const payload = await apiFetch(`/api/submissions/${id}/refresh`, {
        method: "POST"
      });

      await this.loadSubmissions();

      if (options.stayOnDetail || getRoute().submissionId === id) {
        this.state.currentSubmission = payload.job;
        this.submissionDetail.update(payload.job);
      }
    } catch (error) {
      this.setMessage(error.message);
    } finally {
      if (options.button) {
        setButtonLabel(options.button, options.button.dataset.label || "Refresh", false, options.button.dataset.icon || "refresh");
      }
      this.setBusy(false);
    }
  }

  openProcedureDialog(procedure = null) {
    this.procedureDialog.open(procedure);
  }

  async openExecutionDialog() {
    if (!this.state.procedures.length) {
      await this.loadProcedures();
    }

    if (!this.state.procedures.length) {
      this.setMessage("Create a procedure first.");
      return;
    }

    this.executionDialog.open(this.state.procedures);
  }

  async startExecution(id, options = {}) {
    try {
      this.setBusy(true);
      this.setMessage("");
      if (options.button) {
        setButtonLabel(options.button, options.button.dataset.label || "Start", true, options.button.dataset.icon || "start");
      }
      const payload = await apiFetch(`/api/procedure-executions/${id}/start`, {
        method: "POST"
      });

      await this.loadExecutions();
      this.state.currentExecution = payload.execution;
      this.procedureExecutionDetail.update(payload.execution);
    } catch (error) {
      this.setMessage(error.message);
    } finally {
      if (options.button) {
        setButtonLabel(options.button, options.button.dataset.label || "Start", false, options.button.dataset.icon || "start");
      }
      this.setBusy(false);
    }
  }

  async refreshExecution(id, options = {}) {
    try {
      this.setBusy(true);
      this.setMessage("");
      if (options.button) {
        setButtonLabel(options.button, options.button.dataset.label || "Refresh", true, options.button.dataset.icon || "refresh");
      }
      const payload = await apiFetch(`/api/procedure-executions/${id}/refresh`, {
        method: "POST"
      });

      await this.loadExecutions();
      this.state.currentExecution = payload.execution;
      this.procedureExecutionDetail.update(payload.execution);
    } catch (error) {
      this.setMessage(error.message);
    } finally {
      if (options.button) {
        setButtonLabel(options.button, options.button.dataset.label || "Refresh", false, options.button.dataset.icon || "refresh");
      }
      this.setBusy(false);
    }
  }

  async refreshAll() {
    const refreshLabel = this.refreshAllButton.dataset.label || "Refresh All";

    try {
      this.setBusy(true);
      this.setMessage("");
      setButtonLabel(this.refreshAllButton, refreshLabel, true, "refresh");
      const payload = await apiFetch("/api/submissions/refresh-all", {
        method: "POST"
      });

      this.state.submissions = payload.jobs;
      this.submissionsList.update(this.state.submissions);
      const route = getRoute();

      if (route.submissionId) {
        await this.loadSubmission(route.submissionId);
      }

      const refreshableCount = this.state.submissions.filter((job) => job.canRefresh).length;
      setButtonLabel(this.refreshAllButton, refreshableCount ? `Refresh All (${refreshableCount})` : "Refresh All", false, "refresh");
    } catch (error) {
      this.setMessage(error.message);
    } finally {
      const refreshableCount = this.state.submissions.filter((job) => job.canRefresh).length;
      setButtonLabel(this.refreshAllButton, refreshableCount ? `Refresh All (${refreshableCount})` : refreshLabel, false, "refresh");
      this.setBusy(false);
    }
  }

  startThemeSchedule() {
    applyTheme(getAutomaticTheme());
    window.setInterval(() => {
      applyTheme(getAutomaticTheme());
    }, 60 * 1000);
  }

  startExecutionRefreshSchedule() {
    window.setInterval(async () => {
      if (this.state.busy || this.executionRefreshInFlight || document.hidden) {
        return;
      }

      const route = getRoute();
      if (route.page !== "procedure-executions" && route.page !== "execution-detail") {
        return;
      }

      this.executionRefreshInFlight = true;

      try {
        await this.loadExecutions();

        if (route.page === "execution-detail") {
          const execution = await this.loadExecution(route.executionId);
          this.procedureExecutionDetail.update(execution);
        }
      } catch (error) {
        // Keep background polling silent so transient errors do not interrupt manual work.
      } finally {
        this.executionRefreshInFlight = false;
      }
    }, 15 * 1000);
  }

  async renderRoute() {
    const route = getRoute();

    try {
      await this.ensureCapabilities();
    } catch (error) {
      this.setMessage(error.message);
    }

    if (isPlatformRoutePage(route.page) && !this.state.capabilities.platform) {
      this.unavailablePage.update("platform");
      setChildren(this.pageHost, [this.unavailablePage.el]);
      this.renderChrome();
      return;
    }

    if (isBatchRoutePage(route.page) && !this.state.capabilities.batch) {
      this.unavailablePage.update("batch");
      setChildren(this.pageHost, [this.unavailablePage.el]);
      this.renderChrome();
      return;
    }

    try {
      if (route.page === "submissions") {
        await this.loadSubmissions();
        setChildren(this.pageHost, [this.manualForm.el, this.submissionsList.el]);
      } else if (route.page === "submission-detail") {
        const submission = await this.loadSubmission(route.submissionId);
        this.submissionDetail.update(submission);
        setChildren(this.pageHost, [this.submissionDetail.el]);
      } else if (route.page === "templated-submissions") {
        setChildren(this.pageHost, [this.templatedForm.el]);
      } else if (route.page === "procedures") {
        await this.loadProcedures();
        setChildren(this.pageHost, [this.proceduresPage.el]);
      } else if (route.page === "procedure-executions") {
        await Promise.all([this.loadProcedures(), this.loadExecutions()]);
        this.procedureExecutionsPage.update(this.state.executions);
        setChildren(this.pageHost, [this.procedureExecutionsPage.el]);
      } else if (route.page === "execution-detail") {
        await Promise.all([this.loadProcedures(), this.loadExecutions()]);
        const cached = this.state.executions.find((item) => item.id === route.executionId);
        const execution = cached || (await this.loadExecution(route.executionId));
        this.procedureExecutionDetail.update(execution);
        setChildren(this.pageHost, [this.procedureExecutionDetail.el]);
      } else if (route.page === "platform-home") {
        await Promise.all([
          this.loadPlatformConfig(),
          this.loadPlatformCompanies(this.state.platformCompanySearch),
          this.loadPlatformReviews(this.state.platformReviewBookType),
          this.loadPlatformWorkflowRuns(),
          this.loadPlatformAIBatchJobs(),
          this.loadPlatformPositions(this.state.platformPositionsBookType)
        ]);
        this.platformHomePage.update(this.state);
        setChildren(this.pageHost, [this.platformHomePage.el]);
      } else if (route.page === "platform-companies") {
        await this.loadPlatformCompanies(this.state.platformCompanySearch);
        setChildren(this.pageHost, [this.platformCompaniesPage.el]);
      } else if (route.page === "platform-company-detail") {
        await this.loadPlatformCompany(route.companyId);
        setChildren(this.pageHost, [this.platformCompanyDetailPage.el]);
      } else if (route.page === "platform-reviews") {
        await this.loadPlatformReviews(this.state.platformReviewBookType);
        setChildren(this.pageHost, [this.platformReviewsPage.el]);
      } else if (route.page === "platform-review-detail") {
        await this.loadPlatformReview(route.reviewId);
        setChildren(this.pageHost, [this.platformReviewDetailPage.el]);
      } else if (route.page === "platform-workflow-runs") {
        await this.loadPlatformWorkflowRuns();
        setChildren(this.pageHost, [this.platformWorkflowRunsPage.el]);
      } else if (route.page === "platform-workflow-run-detail") {
        await this.loadPlatformWorkflowRun(route.workflowRunId);
        setChildren(this.pageHost, [this.platformWorkflowRunDetailPage.el]);
      } else if (route.page === "platform-ai-batch-jobs") {
        await this.loadPlatformAIBatchJobs();
        setChildren(this.pageHost, [this.platformAIBatchJobsPage.el]);
      } else if (route.page === "platform-ai-batch-job-detail") {
        await this.loadPlatformAIBatchJob(route.aiBatchJobId);
        setChildren(this.pageHost, [this.platformAIBatchJobDetailPage.el]);
      } else if (route.page === "platform-capital-allocations") {
        await this.loadPlatformCapitalAllocations();
        setChildren(this.pageHost, [this.platformCapitalAllocationsPage.el]);
      } else if (route.page === "platform-capital-allocation-detail") {
        await this.loadPlatformCapitalAllocation(route.capitalAllocationId);
        setChildren(this.pageHost, [this.platformCapitalAllocationDetailPage.el]);
      } else if (route.page === "platform-positions") {
        await this.loadPlatformPositions(this.state.platformPositionsBookType);
        setChildren(this.pageHost, [this.platformPositionsPage.el]);
      } else if (route.page === "platform-config") {
        await this.loadPlatformConfig();
        setChildren(this.pageHost, [this.platformConfigPage.el]);
      } else if (route.page === "platform-config-snapshot-detail") {
        await this.loadPlatformConfigSnapshot(route.configSnapshotId);
        setChildren(this.pageHost, [this.platformConfigSnapshotDetailPage.el]);
      } else if (route.page === "platform-overrides") {
        await this.loadPlatformOverrides();
        setChildren(this.pageHost, [this.platformOverridesPage.el]);
      } else {
        this.welcomePage.update(this.state.capabilities);
        setChildren(this.pageHost, [this.welcomePage.el]);
      }
    } catch (error) {
      this.setMessage(error.message);

      if (route.page === "platform-companies") {
        this.platformCompaniesPage.setError(error.message);
        setChildren(this.pageHost, [this.platformCompaniesPage.el]);
      } else {
        this.welcomePage.update(this.state.capabilities);
        setChildren(this.pageHost, [this.welcomePage.el]);
      }
    }

    this.renderChrome();
  }
}

const app = new App();

mount(document.getElementById("app-root"), app);
app.renderRoute();
