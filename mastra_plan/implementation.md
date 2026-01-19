# Implementation Plan

## 1. Setup

See [Setup Guide](./setup.md).

## 2. Agent Definition

We define a simple Agent in Mastra that uses the Cinder tools.

```typescript
import { Agent } from "@mastra/core";
import { crawlTool, scrapeTool, searchTool } from "./tools";

export const cinderAgent = new Agent({
  name: "Cinder Research Agent",
  instructions: "You are a research assistant. Use the provided tools to find information.",
  tools: {
    crawl: crawlTool,
    scrape: scrapeTool,
    search: searchTool,
  },
  model: {
    provider: "OPEN_AI",
    name: "gpt-4",
  },
});
```

### 3. Integration with Cinder

We will use the **Cinder Go Backend** for all heavy lifting.

-   **Endpoint**: `POST /v1/scrape` (Synchronous) or `POST /v1/crawl` (Asynchronous).
-   **Architecture**: Cinder uses a **Singleton Browser** pattern and runs in **Monolith Mode** on Leapcell.
    -   This means we don't need to manage a separate Worker deployment.
    -   We just call the API, and it handles queue processing internally.
-   **Client**: The Mastra Agent will use a simple Fetch wrapper to talk to Cinder.

```typescript
const scrapeResult = await fetch('https://cinder-api.leapcell.dev/v1/scrape', {
  method: 'POST',
  body: JSON.stringify({ url: "https://example.com", render: true, waitFor: 1000 })
});
```

### 4. Workflow Definition

(Optional) For complex tasks, we can define a Workflow.

```typescript
// ... workflow code ...
```
