import { ObjectId } from "mongodb";

import { getDb } from "../db.js";
import { config } from "../config.js";

function getCollection(db) {
  return db.collection(config.mongodb.procedureExecutionsCollectionName);
}

function normalizeExecution(execution) {
  if (!execution) {
    return null;
  }

  return {
    ...execution,
    id: execution._id.toString()
  };
}

export async function listProcedureExecutions() {
  const db = await getDb();
  const collection = getCollection(db);
  const executions = await collection.find({}).sort({ updatedAt: -1, createdAt: -1 }).toArray();

  return executions.map(normalizeExecution);
}

export async function createProcedureExecution(execution) {
  const db = await getDb();
  const collection = getCollection(db);
  const result = await collection.insertOne(execution);

  return getProcedureExecutionById(result.insertedId.toString());
}

export async function getProcedureExecutionById(id) {
  if (!ObjectId.isValid(id)) {
    return null;
  }

  const db = await getDb();
  const collection = getCollection(db);
  const execution = await collection.findOne({ _id: new ObjectId(id) });

  return normalizeExecution(execution);
}

export async function updateProcedureExecution(id, updates) {
  if (!ObjectId.isValid(id)) {
    return null;
  }

  const db = await getDb();
  const collection = getCollection(db);

  await collection.updateOne(
    { _id: new ObjectId(id) },
    {
      $set: {
        ...updates,
        updatedAt: new Date()
      }
    }
  );

  return getProcedureExecutionById(id);
}
