import { ObjectId } from "mongodb";

import { getDb } from "../db.js";
import { config } from "../config.js";

function getCollection(db) {
  return db.collection(config.mongodb.collectionName);
}

function normalizeJob(job) {
  if (!job) {
    return null;
  }

  return {
    ...job,
    id: job._id.toString()
  };
}

export async function listJobs() {
  const db = await getDb();
  const collection = getCollection(db);
  const jobs = await collection.find({}).sort({ createdAt: -1 }).toArray();

  return jobs.map(normalizeJob);
}

export async function createJob(job) {
  const db = await getDb();
  const collection = getCollection(db);
  const result = await collection.insertOne(job);

  return getJobById(result.insertedId.toString());
}

export async function createJobs(jobs) {
  if (!jobs.length) {
    return [];
  }

  const db = await getDb();
  const collection = getCollection(db);
  const result = await collection.insertMany(jobs);
  const ids = Object.values(result.insertedIds).map((value) => value.toString());

  return getJobsByIds(ids);
}

export async function getJobById(id) {
  if (!ObjectId.isValid(id)) {
    return null;
  }

  const db = await getDb();
  const collection = getCollection(db);
  const job = await collection.findOne({ _id: new ObjectId(id) });

  return normalizeJob(job);
}

export async function getJobsByIds(ids) {
  const validIds = ids.filter((id) => ObjectId.isValid(id)).map((id) => new ObjectId(id));

  if (!validIds.length) {
    return [];
  }

  const db = await getDb();
  const collection = getCollection(db);
  const jobs = await collection.find({ _id: { $in: validIds } }).toArray();
  const jobsById = new Map(jobs.map((job) => [job._id.toString(), normalizeJob(job)]));

  return ids.map((id) => jobsById.get(id)).filter(Boolean);
}

export async function listJobsByBatchId(batchId) {
  const db = await getDb();
  const collection = getCollection(db);
  const jobs = await collection.find({ batchId }).sort({ createdAt: -1 }).toArray();

  return jobs.map(normalizeJob);
}

export async function listJobsByStatuses(statuses) {
  const db = await getDb();
  const collection = getCollection(db);
  const jobs = await collection.find({ status: { $in: statuses } }).sort({ createdAt: -1 }).toArray();

  return jobs.map(normalizeJob);
}

export async function updateJob(id, updates) {
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

  return getJobById(id);
}
