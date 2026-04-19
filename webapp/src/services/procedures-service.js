import {
  createProcedure,
  getProcedureById,
  listProcedures,
  updateProcedure
} from "../repositories/procedures-repository.js";
import { DEFAULT_MODEL, normalizeModelName, normalizeReasoningEffort } from "../shared/openai-models.js";

function normalizeProcedureSteps(rawSteps) {
  const steps = Array.isArray(rawSteps) ? rawSteps : [];
  const normalizedSteps = steps
    .map((step, index) => {
      const prompt = typeof step?.prompt === "string" ? step.prompt.trim() : "";

      if (!prompt) {
        return null;
      }

      const model = normalizeModelName(step?.model || DEFAULT_MODEL);

      return {
        id: `step-${index + 1}`,
        stepNumber: 0,
        prompt,
        model,
        reasoningEffort: normalizeReasoningEffort(model, step?.reasoningEffort)
      };
    })
    .filter(Boolean)
    .map((step, index) => ({
      ...step,
      id: `step-${index + 1}`,
      stepNumber: index + 1
    }));

  if (!normalizedSteps.length) {
    throw new Error("Please add at least one procedure step with a prompt.");
  }

  return normalizedSteps;
}

function toViewModel(procedure) {
  if (!procedure) {
    return null;
  }

  return {
    ...procedure,
    steps: Array.isArray(procedure.steps) ? procedure.steps : [],
    stepCount: Array.isArray(procedure.steps) ? procedure.steps.length : 0,
    createdAtLabel: procedure.createdAt ? new Date(procedure.createdAt).toLocaleString() : "Unknown",
    updatedAtLabel: procedure.updatedAt ? new Date(procedure.updatedAt).toLocaleString() : "Unknown"
  };
}

export async function getProceduresForList() {
  const procedures = await listProcedures();
  return procedures.map(toViewModel);
}

export async function getProcedureDetails(id) {
  const procedure = await getProcedureById(id);
  return toViewModel(procedure);
}

export async function createProcedureDefinition({ name, steps }) {
  const trimmedName = typeof name === "string" ? name.trim() : "";

  if (!trimmedName) {
    throw new Error("Procedure name is required.");
  }

  const now = new Date();
  const procedure = await createProcedure({
    name: trimmedName,
    steps: normalizeProcedureSteps(steps),
    createdAt: now,
    updatedAt: now
  });

  return toViewModel(procedure);
}

export async function updateProcedureDefinition(id, { name, steps }) {
  const trimmedName = typeof name === "string" ? name.trim() : "";

  if (!trimmedName) {
    throw new Error("Procedure name is required.");
  }

  const existingProcedure = await getProcedureById(id);

  if (!existingProcedure) {
    return null;
  }

  const procedure = await updateProcedure(id, {
    name: trimmedName,
    steps: normalizeProcedureSteps(steps)
  });

  return toViewModel(procedure);
}
