# Implementation Guide

Follow these steps to build the Mastra MCP server.

## Prerequisites

- Node.js 20+
- Cinder API running (locally or remotely)

## Step 1: Initialize Mastra Project

Create a new directory `mastra` in the root of the repository and initialize a Mastra project.

```bash
mkdir mastra
cd mastra
npm init -y
npm install typescript @types/node mastra @mastra/core @mastra/mcp zod
npx tsc --init
```

## Step 2: Project Structure

Ensure your `mastra` directory looks like this:

```
mastra/
├── src/
│   ├── mastra/
│   │   ├── tools/
│   │   │   ├── index.ts
│   │   │   ├── scrape.ts
│   │   │   ├── search.ts
│   │   │   └── crawl.ts
│   │   ├── index.ts (Entry point)
│   │   └── server.ts (MCP Server config)
├── package.json
└── tsconfig.json
```

## Step 3: Implement Tools

Create the tool files in `src/mastra/tools/`.

### `src/mastra/tools/scrape.ts`

```typescript
import { createTool } from "@mastra/core/tools";
import { z } from "zod";

const CINDER_API_URL = process.env.CINDER_API_URL || "http://localhost:8080";

export const scrapeTool = createTool({
  id: "cinder_scrape",
  description: "Scrape content from a URL",
  inputSchema: z.object({
    url: z.string().url(),
    mode: z.enum(["smart", "static", "dynamic"]).optional().default("smart"),
  }),
  execute: async ({ context }) => {
    const response = await fetch(`${CINDER_API_URL}/v1/scrape`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(context),
    });
    return await response.json();
  },
});
```

_Repeat similar implementation for `search` and `crawl` tools based on [Tools Specification](./tools.md)._

## Step 4: Configure MCP Server

Create `src/mastra/server.ts` to aggregate tools and export the server.

```typescript
import { MCPServer } from "@mastra/mcp";
import { scrapeTool } from "./tools/scrape";
import { searchTool } from "./tools/search";
import { crawlTool, crawlStatusTool } from "./tools/crawl";

export const cinderMcpServer = new MCPServer({
  name: "cinder-mcp",
  version: "1.0.0",
  tools: {
    scrapeTool,
    searchTool,
    crawlTool,
    crawlStatusTool,
  },
});
```

## Step 5: Entry Point

Create `src/mastra/index.ts` to start the server.

```typescript
import { cinderMcpServer } from "./server";

// Start stdio server for local usage (e.g. with Claude Desktop)
cinderMcpServer.startStdio();
```

## Step 6: Build Script

Update `package.json` to include a build script.

```json
"scripts": {
  "build": "tsc",
  "start": "node dist/mastra/index.js"
}
```
