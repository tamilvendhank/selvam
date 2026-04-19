import { el, mount, setChildren } from "/vendor/redom.es.min.js";
import { marked } from "/vendor-marked/marked.esm.js";
import {
  DEFAULT_MODEL,
  MODEL_SUGGESTIONS,
  getReasoningProfile,
  normalizeModelName,
  normalizeReasoningEffort
} from "/shared/openai-models.js";

const NAV_ITEMS = [
  { path: "/", label: "Welcome", icon: "welcome" },
  { path: "/submissions", label: "Submissions", icon: "submissions" },
  { path: "/templated-submissions", label: "Templated Submissions", icon: "templated" },
  { path: "/procedures", label: "Procedures", icon: "procedures" },
  { path: "/procedure-executions", label: "Procedure Executions", icon: "executions" }
];
const ICONS = {
  welcome: "fa-solid fa-house",
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
    throw new Error(payload?.error || "Request failed.");
  }

  return payload;
}

function pathForSubmission(id) {
  return `/submissions/${id}`;
}

function pathForExecution(id) {
  return `/procedure-executions/${id}`;
}

function getRoute() {
  const pathname = window.location.pathname;
  const submissionMatch = pathname.match(/^\/submissions\/([^/]+)$/);
  const executionMatch = pathname.match(/^\/procedure-executions\/([^/]+)$/);

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

  return { page: "welcome", submissionId: null };
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
    this.links = NAV_ITEMS.map((item) => new NavLink(app, item));
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
        el("div", el("p.eyebrow", textWithIcon("procedure", "Workspace")), el("h2.sidebar-title", "OpenAI Batch App")),
        this.closeButton
      ),
      this.nav
    );
  }

  update(currentPath, isOpen) {
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

class WelcomePage {
  constructor() {
    this.el = el(
      ".panel.welcome-panel",
      el("p.eyebrow", textWithIcon("welcome", "Welcome")),
      el("h1.detail-title", "Batch submissions, one workspace."),
      el(
        "p.hero-text",
        "Use the navigation on the left to create manual submissions, build templated submissions from JSON data, define procedures, and run multi-step executions in one place."
      )
    );
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
      submissions: [],
      procedures: [],
      executions: [],
      currentSubmission: null,
      currentExecution: null,
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
    this.welcomePage = new WelcomePage();
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

  renderChrome() {
    const route = getRoute();
    const currentPath =
      route.page === "submission-detail"
        ? "/submissions"
        : route.page === "execution-detail"
          ? "/procedure-executions"
          : window.location.pathname;
    const isSubmissionRoute = route.page === "submissions" || route.page === "submission-detail";
    this.sidebar.update(currentPath, this.state.sidebarOpen);
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
      if (route.page === "submissions" || route.page === "submission-detail") {
        await this.loadSubmissions();
      } else if (route.page === "procedures") {
        await this.loadProcedures();
      } else if (route.page === "procedure-executions" || route.page === "execution-detail") {
        await Promise.all([this.loadProcedures(), this.loadExecutions()]);
      }
    } catch (error) {
      this.setMessage(error.message);
    }

    if (route.page === "submissions") {
      setChildren(this.pageHost, [this.manualForm.el, this.submissionsList.el]);
    } else if (route.page === "submission-detail") {
      try {
        const submission = await this.loadSubmission(route.submissionId);
        this.submissionDetail.update(submission);
      } catch (error) {
        this.setMessage(error.message);
      }

      setChildren(this.pageHost, [this.submissionDetail.el]);
    } else if (route.page === "templated-submissions") {
      setChildren(this.pageHost, [this.templatedForm.el]);
    } else if (route.page === "procedures") {
      setChildren(this.pageHost, [this.proceduresPage.el]);
    } else if (route.page === "procedure-executions") {
      this.procedureExecutionsPage.update(this.state.executions);
      setChildren(this.pageHost, [this.procedureExecutionsPage.el]);
    } else if (route.page === "execution-detail") {
      try {
        const cached = this.state.executions.find((item) => item.id === route.executionId);
        const execution = cached || (await this.loadExecution(route.executionId));
        this.procedureExecutionDetail.update(execution);
      } catch (error) {
        this.setMessage(error.message);
      }

      setChildren(this.pageHost, [this.procedureExecutionDetail.el]);
    } else {
      setChildren(this.pageHost, [this.welcomePage.el]);
    }

    this.renderChrome();
  }
}

const app = new App();

mount(document.getElementById("app-root"), app);
app.renderRoute();
