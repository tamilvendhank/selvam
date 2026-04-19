export const DEFAULT_MODEL = "gpt-4.1-mini";

export const MODEL_SUGGESTIONS = [
  "gpt-5.4",
  "gpt-5.4-mini",
  "gpt-5.4-nano",
  "gpt-5.4-pro",
  "gpt-5.2",
  "gpt-5.2-pro",
  "gpt-5.1",
  "gpt-5",
  "gpt-5-mini",
  "gpt-5-nano",
  "gpt-5-pro",
  "gpt-4.1",
  "gpt-4.1-mini",
  "gpt-4.1-nano",
  "gpt-4o",
  "gpt-4o-mini",
  "o4-mini",
  "o3"
];

function normalizeLower(value) {
  return String(value || "").trim().toLowerCase();
}

export function normalizeModelName(model) {
  const trimmed = String(model || "").trim();
  return trimmed || DEFAULT_MODEL;
}

export function getReasoningProfile(model) {
  const normalized = normalizeLower(model);

  if (!normalized) {
    return {
      supported: true,
      options: ["none", "minimal", "low", "medium", "high", "xhigh"],
      defaultValue: "medium"
    };
  }

  if (normalized.includes("pro")) {
    return {
      supported: true,
      options: ["high"],
      defaultValue: "high"
    };
  }

  if (normalized.startsWith("gpt-5.1")) {
    return {
      supported: true,
      options: ["none", "low", "medium", "high"],
      defaultValue: "none"
    };
  }

  if (
    normalized.startsWith("gpt-5.2") ||
    normalized.startsWith("gpt-5.3") ||
    normalized.startsWith("gpt-5.4") ||
    normalized.startsWith("gpt-5.5") ||
    normalized.startsWith("gpt-6")
  ) {
    return {
      supported: true,
      options: ["none", "minimal", "low", "medium", "high", "xhigh"],
      defaultValue: "medium"
    };
  }

  if (normalized.startsWith("gpt-5") || normalized.startsWith("o")) {
    return {
      supported: true,
      options: ["minimal", "low", "medium", "high"],
      defaultValue: "medium"
    };
  }

  return {
    supported: false,
    options: [],
    defaultValue: null
  };
}

export function normalizeReasoningEffort(model, reasoningEffort) {
  const profile = getReasoningProfile(model);

  if (!profile.supported) {
    return null;
  }

  const normalized = normalizeLower(reasoningEffort);

  if (normalized && profile.options.includes(normalized)) {
    return normalized;
  }

  return profile.defaultValue;
}
