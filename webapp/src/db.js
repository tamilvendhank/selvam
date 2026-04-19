import { MongoClient } from "mongodb";

import { config } from "./config.js";

let client;
let clientPromise;

export async function getDb() {
  if (!clientPromise) {
    client = new MongoClient(config.mongodb.uri);
    clientPromise = client.connect();
  }

  const connectedClient = await clientPromise;
  return connectedClient.db(config.mongodb.dbName);
}

export async function closeDb() {
  if (!client) {
    return;
  }

  await client.close();
  client = undefined;
  clientPromise = undefined;
}
