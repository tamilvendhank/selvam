# OpenAI Batch Web App

A small Node.js app that lets you:

- submit a query from the browser
- store the query in MongoDB on localhost
- send it to OpenAI using the Batch API with the modern `/v1/responses` endpoint
- see every submitted job in a list
- open completed jobs to view results
- manually refresh incomplete jobs

## Requirements

- Node.js 22+
- MongoDB running locally
- an OpenAI API key

## Setup

1. Copy `.env.example` to `.env`
2. Fill in `OPENAI_API_KEY`
3. Install dependencies:

```bash
npm install
```

4. Start the app:

```bash
npm start
```

5. Open [http://localhost:3000](http://localhost:3000)

## Notes

- Each submitted query is stored first in MongoDB, then sent to OpenAI as a single-request batch job.
- The app uses the Responses API through Batch because that is the recommended approach for new OpenAI projects.
- Batch jobs are asynchronous and can take some time to complete, so incomplete jobs expose a refresh button in the UI.
