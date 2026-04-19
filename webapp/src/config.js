import dotenv from "dotenv";

dotenv.config();

function requireEnv(name) {
  const value = process.env[name];

  if (!value) {
    throw new Error(`Missing required environment variable: ${name}`);
  }

  return value;
}

export const config = {
  port: Number(process.env.PORT || 3000),
  mongodb: {
    uri: process.env.MONGODB_URI || "mongodb://127.0.0.1:27017",
    dbName: process.env.MONGODB_DB_NAME || "openai_batch_webapp",
    collectionName: "query_jobs",
    proceduresCollectionName: "procedures",
    procedureExecutionsCollectionName: "procedure_executions"
  },
  openai: {
    apiKey: requireEnv("OPENAI_API_KEY"),
    model: process.env.OPENAI_MODEL || "gpt-4.1-mini",
    responseInstructions:
      process.env.OPENAI_RESPONSE_INSTRUCTIONS ||
      "Answer the user's query clearly and concisely.",
    batchEndpoint: "/v1/responses",
    completionWindow: "24h"
  }
};
