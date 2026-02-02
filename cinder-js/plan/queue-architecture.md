# Queue Architecture (BullMQ)

> **Purpose:** Document the async job queue design using BullMQ  
> **Replaces:** Go Asynq implementation  
> **Last Updated:** 2026-02-02

---

## Table of Contents

1. [Overview](#overview)
2. [BullMQ vs Asynq Mapping](#bullmq-vs-asynq-mapping)
3. [Queue Design](#queue-design)
4. [Worker Configuration](#worker-configuration)
5. [Monolith Pattern Implementation](#monolith-pattern-implementation)
6. [Graceful Shutdown](#graceful-shutdown)
7. [Monitoring & Observability](#monitoring--observability)

---

## Overview

### Purpose

The queue system handles async crawling operations:
1. Accept crawl request → return job ID immediately
2. Process crawl in background
3. Store result for later retrieval
4. Handle retries and failures

### Architecture

```
┌───────────────────────────────────────────────────────────┐
│                        CLIENT                             │
└─────────────────────────┬─────────────────────────────────┘
                          │
              ┌───────────▼───────────┐
              │    POST /v1/crawl     │
              │    (Hono Endpoint)    │
              └───────────┬───────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│                      PRODUCER                            │
│                                                          │
│  1. Validate request (Valibot)                          │
│  2. Generate job ID (UUID)                              │
│  3. Add job to queue                                    │
│  4. Return 202 + job ID                                 │
└─────────────────────────────┬───────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────┐
│                       REDIS                              │
│                                                          │
│  BullMQ Queues:                                         │
│  ├── scrape:waiting     (pending jobs)                  │
│  ├── scrape:active      (in-progress)                   │
│  ├── scrape:completed   (finished)                      │
│  ├── scrape:failed      (attempts exhausted)            │
│  └── scrape:delayed     (scheduled for later)           │
│                                                          │
│  Job Storage:                                            │
│  └── bull:scrape:job:{id} → {data, result, state}       │
└─────────────────────────────┬───────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────┐
│                      WORKER                              │
│                                                          │
│  1. Poll for jobs (configurable interval)               │
│  2. Acquire job lock (atomic)                           │
│  3. Execute scraper service                             │
│  4. Store result                                        │
│  5. Mark complete or failed                             │
└─────────────────────────────────────────────────────────┘
```

---

## BullMQ vs Asynq Mapping

### Core Concepts

| Asynq (Go) | BullMQ (JS)        | Notes              |
| ---------- | ------------------ | ------------------ |
| Task       | Job                | Equivalent concept |
| Server     | Worker             | Processes jobs     |
| Client     | Queue              | Enqueues jobs      |
| ServeMux   | Processor function | Handler routing    |
| Inspector  | Queue.getJobs()    | Job inspection     |

### Configuration Mapping

**Go (Asynq):**
```go
srv := asynq.NewServer(
    redisOpt,
    asynq.Config{
        Concurrency: 10,
        Queues: map[string]int{
            "critical": 6,
            "default":  3,
            "low":      1,
        },
        TaskCheckInterval: 1 * time.Second,
    },
)
```

**JS (BullMQ):**
```typescript
// Queue creation (for producing)
const scrapeQueue = new Queue('scrape', {
  connection: redisConnection,
  defaultJobOptions: {
    attempts: 3,
    backoff: {
      type: 'exponential',
      delay: 1000,
    },
    removeOnComplete: 100, // Keep last 100 completed
    removeOnFail: 50,      // Keep last 50 failed
  },
});

// Worker creation (for consuming)
const worker = new Worker('scrape', processor, {
  connection: redisConnection,
  concurrency: 10,
  limiter: {
    max: 100,
    duration: 60000, // Max 100 jobs per minute
  },
});
```

### Priority Mapping

BullMQ uses numeric priority (lower = higher priority):

| Asynq Queue | Asynq Weight | BullMQ Priority |
| ----------- | ------------ | --------------- |
| critical    | 6            | 1               |
| default     | 3            | 5               |
| low         | 1            | 10              |

**Usage:**
```typescript
// Enqueue with priority
await scrapeQueue.add('scrape', payload, { priority: 1 });  // critical
await scrapeQueue.add('scrape', payload, { priority: 5 });  // default
await scrapeQueue.add('scrape', payload, { priority: 10 }); // low
```

---

## Queue Design

### Job Data Schema

```typescript
// Using Valibot for validation
import { object, string, boolean, optional, InferOutput } from 'valibot';

const CrawlJobSchema = object({
  url: string(),
  render: optional(boolean()),
  maxDepth: optional(number()),
  maxPages: optional(number()),
});

type CrawlJob = InferOutput<typeof CrawlJobSchema>;

// Job payload structure
interface JobPayload {
  id: string;           // UUID
  url: string;          // Root URL to crawl
  render: boolean;      // Use Playwright
  createdAt: number;    // Unix timestamp
}

// Job result structure
interface JobResult {
  urlsScraped: number;
  markdown: string;
  metadata: Record<string, string>;
  duration: number;     // ms
  errors: string[];
}
```

### Queue Configuration

```typescript
// Queue options
const QUEUE_CONFIG = {
  name: 'scrape',
  
  // Default job options
  defaultJobOptions: {
    // Retry configuration
    attempts: 3,
    backoff: {
      type: 'exponential',
      delay: 1000,  // First retry after 1s, then 2s, 4s
    },
    
    // Cleanup
    removeOnComplete: {
      count: 100,   // Keep last 100 completed jobs
      age: 86400,   // Or jobs completed in last 24h
    },
    removeOnFail: {
      count: 50,    // Keep last 50 failed jobs
      age: 172800,  // Or jobs failed in last 48h
    },
    
    // Timeout
    timeout: 300000, // 5 minutes max per job
  },
};
```

### Job States

```
                       ┌──────────────┐
                       │   waiting    │
                       │   (queued)   │
                       └──────┬───────┘
                              │
                              │ Worker picks up
                              ▼
                       ┌──────────────┐
                       │    active    │
                       │ (processing) │
                       └──────┬───────┘
                              │
            ┌─────────────────┼─────────────────┐
            │                 │                 │
            ▼                 ▼                 ▼
     ┌──────────┐      ┌──────────┐      ┌──────────┐
     │completed │      │  failed  │      │ delayed  │
     │ (done)   │      │ (error)  │      │ (retry)  │
     └──────────┘      └──────────┘      └─────┬────┘
                              ▲                 │
                              │                 │
                              └─────────────────┘
                                (after delay)
```

---

## Worker Configuration

### Basic Worker Setup

```typescript
// Conceptual implementation
import { Worker, Job } from 'bullmq';
import { scraperService } from './services/scraper';

const worker = new Worker(
  'scrape',
  async (job: Job<JobPayload, JobResult>) => {
    const { url, render } = job.data;
    
    // Update progress
    await job.updateProgress(10);
    
    // Execute scrape
    const mode = render ? 'dynamic' : 'smart';
    const result = await scraperService.scrape(url, mode);
    
    await job.updateProgress(100);
    
    return {
      urlsScraped: 1,
      markdown: result.markdown,
      metadata: result.metadata,
      duration: Date.now() - job.timestamp,
      errors: [],
    };
  },
  {
    connection: redisConnection,
    concurrency: 10,
    
    // Stalled job detection
    lockDuration: 60000,      // Job lock expires after 60s
    stalledInterval: 30000,   // Check for stalled jobs every 30s
    maxStalledCount: 2,       // Retry stalled jobs up to 2 times
  }
);
```

### Event Handling

```typescript
// Worker events
worker.on('completed', (job, result) => {
  logger.info(`Job ${job.id} completed`, { 
    url: job.data.url,
    duration: result.duration 
  });
});

worker.on('failed', (job, error) => {
  logger.error(`Job ${job?.id} failed`, { 
    url: job?.data.url,
    error: error.message,
    attempts: job?.attemptsMade 
  });
});

worker.on('stalled', (jobId) => {
  logger.warn(`Job ${jobId} stalled - will be retried`);
});

worker.on('error', (error) => {
  logger.error('Worker error', { error: error.message });
});
```

### Rate Limiting

```typescript
const worker = new Worker('scrape', processor, {
  connection: redisConnection,
  concurrency: 10,
  
  // Rate limit: max 100 jobs per minute
  limiter: {
    max: 100,
    duration: 60000,
  },
  
  // Per-job rate limiting (optional)
  limiter: {
    max: 1,              // 1 job per host
    duration: 1000,      // per second
    groupKey: 'host',    // Group by job.data.host
  },
});
```

---

## Monolith Pattern Implementation

### Challenge

Running API server + queue worker in the same container requires careful orchestration.

**Go Approach (Current):**
```go
// Go uses goroutines - lightweight threads
go func() {
    if err := srv.Run(mux); err != nil {
        log.Fatal("worker error:", err)
    }
}()

// API runs in main goroutine
router.Run(":" + port)
```

**JS Challenge:**
- Node.js is single-threaded
- BullMQ worker can block event loop
- Need worker_threads or careful async design

### Solution: Same-Thread Approach

For I/O-bound scraping, the worker can run in the same event loop:

```typescript
// Main entry point
async function main() {
  // Initialize shared resources
  const redis = createRedisConnection();
  const browser = await initBrowser();
  const scraperService = new ScraperService(browser);
  
  // Create queue + worker
  const queue = new Queue('scrape', { connection: redis });
  const worker = new Worker('scrape', createProcessor(scraperService), {
    connection: redis,
    concurrency: 10,
  });
  
  // Create Hono app with queue access
  const app = createApp({ queue, scraperService });
  
  // Setup shutdown handling
  setupGracefulShutdown({ worker, queue, browser });
  
  // Start HTTP server
  console.log('Starting server on :8080');
  Bun.serve({
    port: 8080,
    fetch: app.fetch,
  });
}

main().catch(console.error);
```

### Alternative: Worker Thread Approach

For CPU-heavy processing (not typical for scraping):

```typescript
// main.ts
import { Worker as NodeWorker } from 'worker_threads';

async function main() {
  // Start worker in separate thread
  const workerThread = new NodeWorker('./worker-thread.js');
  
  workerThread.on('message', (msg) => {
    console.log('Worker message:', msg);
  });
  
  workerThread.on('error', (err) => {
    console.error('Worker error:', err);
  });
  
  // Start API server in main thread
  const app = createApp();
  Bun.serve({ port: 8080, fetch: app.fetch });
}

// worker-thread.ts (separate file)
import { Worker } from 'bullmq';

const worker = new Worker('scrape', processor, options);
console.log('Worker thread started');
```

**Recommendation:** Start with same-thread approach (simpler). Only move to worker threads if CPU becomes bottleneck.

---

## Graceful Shutdown

### Requirements

On `SIGTERM` or `SIGINT`:
1. Stop accepting new HTTP requests
2. Stop accepting new jobs
3. Wait for active jobs to complete (with timeout)
4. Close browser contexts
5. Close Redis connections
6. Exit cleanly

### Implementation

```typescript
function setupGracefulShutdown(deps: {
  worker: Worker;
  queue: Queue;
  browser: Browser;
}) {
  const { worker, queue, browser } = deps;
  
  async function shutdown(signal: string) {
    console.log(`Received ${signal}, shutting down gracefully...`);
    
    try {
      // 1. Stop accepting HTTP requests
      // (Bun.serve handles this automatically on exit)
      
      // 2. Pause worker (stop picking up new jobs)
      await worker.pause(true);
      console.log('Worker paused');
      
      // 3. Wait for active jobs (with timeout)
      const activeJobs = await queue.getActiveCount();
      console.log(`Waiting for ${activeJobs} active jobs...`);
      
      const timeout = 30000; // 30 seconds
      const start = Date.now();
      
      while (Date.now() - start < timeout) {
        const remaining = await queue.getActiveCount();
        if (remaining === 0) break;
        await new Promise(r => setTimeout(r, 1000));
        console.log(`Still waiting for ${remaining} jobs...`);
      }
      
      // 4. Close worker
      await worker.close();
      console.log('Worker closed');
      
      // 5. Close browser
      await browser.close();
      console.log('Browser closed');
      
      // 6. Close queue (and Redis connection)
      await queue.close();
      console.log('Queue closed');
      
      console.log('Graceful shutdown complete');
      process.exit(0);
    } catch (error) {
      console.error('Error during shutdown:', error);
      process.exit(1);
    }
  }
  
  process.on('SIGTERM', () => shutdown('SIGTERM'));
  process.on('SIGINT', () => shutdown('SIGINT'));
}
```

### Leapcell Considerations

Leapcell sends `SIGTERM` before container termination:

1. **Default timeout:** 30 seconds before force kill
2. **Health checks:** Should return unhealthy when shutting down
3. **Drain connections:** Stop accepting new requests immediately

```typescript
// Health check during shutdown
let isShuttingDown = false;

app.get('/health', (c) => {
  if (isShuttingDown) {
    return c.json({ status: 'shutting_down' }, 503);
  }
  return c.json({ status: 'healthy' }, 200);
});

// Set flag on shutdown
process.on('SIGTERM', () => {
  isShuttingDown = true;
  // ... rest of shutdown
});
```

---

## Monitoring & Observability

### BullMQ Metrics

```typescript
// Metrics collection
async function collectQueueMetrics(queue: Queue) {
  const [waiting, active, completed, failed, delayed] = await Promise.all([
    queue.getWaitingCount(),
    queue.getActiveCount(),
    queue.getCompletedCount(),
    queue.getFailedCount(),
    queue.getDelayedCount(),
  ]);
  
  return {
    queue: queue.name,
    waiting,
    active,
    completed,
    failed,
    delayed,
    timestamp: Date.now(),
  };
}

// Periodic metrics logging
setInterval(async () => {
  const metrics = await collectQueueMetrics(scrapeQueue);
  logger.info('Queue metrics', metrics);
}, 60000); // Every minute
```

### Bull Board Dashboard

For development/debugging:

```typescript
import { createBullBoard } from '@bull-board/api';
import { BullMQAdapter } from '@bull-board/api/bullMQAdapter';
import { HonoAdapter } from '@bull-board/hono';

// Add Bull Board to Hono app
const serverAdapter = new HonoAdapter('/admin/queues');
createBullBoard({
  queues: [new BullMQAdapter(scrapeQueue)],
  serverAdapter,
});

app.route('/admin', serverAdapter.registerPlugin());
```

**Security Note:** Protect `/admin` routes in production!

### Job Tracing

```typescript
// Add trace ID to jobs
const traceId = crypto.randomUUID();

await queue.add('scrape', { url, render }, {
  jobId: traceId,  // Use as job ID
});

// In logs
logger.info('Processing job', { 
  jobId: job.id,
  url: job.data.url,
  traceId: job.id // for correlation
});
```

---

## Redis Connection Configuration

### TLS Support (Upstash/Leapcell)

```typescript
import IORedis from 'ioredis';

function createRedisConnection() {
  const redisUrl = process.env.REDIS_URL;
  
  if (!redisUrl) {
    throw new Error('REDIS_URL is required');
  }
  
  // IORedis parses rediss:// automatically for TLS
  const connection = new IORedis(redisUrl, {
    maxRetriesPerRequest: null, // Required for BullMQ
    enableReadyCheck: false,
    
    // TLS options (for rediss://)
    tls: redisUrl.startsWith('rediss://') ? {
      rejectUnauthorized: false, // For some hosted Redis
    } : undefined,
  });
  
  connection.on('error', (err) => {
    console.error('Redis connection error:', err);
  });
  
  connection.on('connect', () => {
    console.log('Redis connected');
  });
  
  return connection;
}
```

### Connection Pooling

BullMQ manages connections efficiently, but for high concurrency:

```typescript
// Separate connections for queue and worker
const queueConnection = createRedisConnection();
const workerConnection = createRedisConnection();

const queue = new Queue('scrape', { connection: queueConnection });
const worker = new Worker('scrape', processor, { connection: workerConnection });
```

---

*Document Version: 1.0.0-draft*  
*Last Updated: 2026-02-02*
