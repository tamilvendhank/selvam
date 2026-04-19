import { ObjectId } from "mongodb";

import { getDb } from "../db.js";
import { config } from "../config.js";

function getCollection(db) {
  return db.collection(config.mongodb.proceduresCollectionName);
}

function normalizeProcedure(procedure) {
  if (!procedure) {
    return null;
  }

  return {
    ...procedure,
    id: procedure._id.toString()
  };
}

export async function listProcedures() {
  const db = await getDb();
  const collection = getCollection(db);
  const procedures = await collection.find({}).sort({ updatedAt: -1, createdAt: -1 }).toArray();

  return procedures.map(normalizeProcedure);
}

export async function createProcedure(procedure) {
  const db = await getDb();
  const collection = getCollection(db);
  const result = await collection.insertOne(procedure);

  return getProcedureById(result.insertedId.toString());
}

export async function getProcedureById(id) {
  if (!ObjectId.isValid(id)) {
    return null;
  }

  const db = await getDb();
  const collection = getCollection(db);
  const procedure = await collection.findOne({ _id: new ObjectId(id) });

  return normalizeProcedure(procedure);
}

export async function updateProcedure(id, updates) {
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

  return getProcedureById(id);
}
